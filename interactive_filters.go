package preditor

import (
	"fmt"
	"github.com/amirrezaask/preditor/components"
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"os"
	"path"
	"path/filepath"
)

type ScoredItem[T any] struct {
	Item  T
	Score int
}

type InteractiveFilter[T any] struct {
	BaseDrawable
	cfg                     *Config
	parent                  *Context
	keymaps                 []Keymap
	List                    components.ListComponent[T]
	UserInputComponent      *components.UserInputComponent
	LastInputWeRanUpdateFor string
	UpdateList              func(list *components.ListComponent[T], input string)
	OpenSelection           func(preditor *Context, t T) error
	ItemRepr                func(item T) string
}

func (i InteractiveFilter[T]) Keymaps() []Keymap {
	return i.keymaps
}

func (i InteractiveFilter[T]) String() string {
	return fmt.Sprintf("InteractiveFilter: %T", *new(T))
}

func NewInteractiveFilter[T any](
	parent *Context,
	cfg *Config,
	updateList func(list *components.ListComponent[T], input string),
	openSelection func(preditor *Context, t T) error,
	repr func(t T) string,
	initialList func() []T,
) *InteractiveFilter[T] {
	ifb := &InteractiveFilter[T]{
		cfg:                cfg,
		parent:             parent,
		keymaps:            []Keymap{makeKeymap[T]()},
		UserInputComponent: components.NewUserInputComponent(),
		UpdateList:         updateList,
		OpenSelection:      openSelection,
		ItemRepr:           repr,
	}
	if initialList != nil {
		iList := initialList()
		ifb.List.Items = iList
	}

	ifb.keymaps = append(ifb.keymaps, MakeInsertionKeys(func(c *Context, b byte) error {
		return ifb.UserInputComponent.InsertCharAtBuffer(b)
	}))
	return ifb
}

func (i *InteractiveFilter[T]) Render(zeroLocation rl.Vector2, maxH float64, maxW float64) {
	if i.LastInputWeRanUpdateFor != string(i.UserInputComponent.UserInput) {
		i.LastInputWeRanUpdateFor = string(i.UserInputComponent.UserInput)
		i.UpdateList(&i.List, string(i.UserInputComponent.UserInput))
	}
	charSize := measureTextSize(i.parent.Font, ' ', i.parent.FontSize, 0)

	//draw input box
	rl.DrawRectangleLines(int32(zeroLocation.X), int32(zeroLocation.Y), int32(maxW), int32(charSize.Y)*2, i.cfg.CurrentThemeColors().StatusBarBackground.ToColorRGBA())
	rl.DrawTextEx(i.parent.Font, string(i.UserInputComponent.UserInput), rl.Vector2{
		X: zeroLocation.X, Y: zeroLocation.Y + charSize.Y/2,
	}, float32(i.parent.FontSize), 0, i.cfg.CurrentThemeColors().Foreground.ToColorRGBA())

	switch i.cfg.CursorShape {
	case CURSOR_SHAPE_OUTLINE:
		rl.DrawRectangleLines(int32(float64(zeroLocation.X)+float64(charSize.X))*int32(i.UserInputComponent.Idx), int32(zeroLocation.Y+charSize.Y/2), int32(charSize.X), int32(charSize.Y), rl.Fade(rl.Red, 0.5))
	case CURSOR_SHAPE_BLOCK:
		rl.DrawRectangle(int32(float64(zeroLocation.X)+float64(charSize.X))*int32(i.UserInputComponent.Idx), int32(zeroLocation.Y+charSize.Y/2), int32(charSize.X), int32(charSize.Y), rl.Fade(rl.Red, 0.5))
	case CURSOR_SHAPE_LINE:
		rl.DrawRectangleLines(int32(float64(zeroLocation.X)+float64(charSize.X))*int32(i.UserInputComponent.Idx), int32(zeroLocation.Y+charSize.Y/2), 2, int32(charSize.Y), rl.Fade(rl.Red, 0.5))
	}

	startOfListY := int32(zeroLocation.Y) + int32(3*(charSize.Y))
	maxLine := int(int32((maxH+float64(zeroLocation.Y))-float64(startOfListY)) / int32(charSize.Y))

	//draw list of items
	for idx, item := range i.List.VisibleView(maxLine) {
		rl.DrawTextEx(i.parent.Font, i.ItemRepr(item), rl.Vector2{
			X: zeroLocation.X, Y: float32(startOfListY) + float32(idx)*charSize.Y,
		}, float32(i.parent.FontSize), 0, i.cfg.CurrentThemeColors().Foreground.ToColorRGBA())
	}
	if len(i.List.Items) > 0 {
		rl.DrawRectangle(int32(zeroLocation.X), int32(int(startOfListY)+(i.List.Selection-i.List.VisibleStart)*int(charSize.Y)), int32(maxW), int32(charSize.Y), rl.Fade(i.cfg.CurrentThemeColors().Selection.ToColorRGBA(), 0.2))
	}
}

func makeKeymap[T any]() Keymap {
	return Keymap{

		Key{K: "f", Control: true}: MakeCommand(func(e *InteractiveFilter[T]) error {
			return e.UserInputComponent.CursorRight(1)
		}),
		Key{K: "v", Control: true}: MakeCommand(func(e *InteractiveFilter[T]) error {
			return e.UserInputComponent.Paste()
		}),
		Key{K: "c", Control: true}: MakeCommand(func(e *InteractiveFilter[T]) error {
			return e.UserInputComponent.Copy()
		}),
		Key{K: "a", Control: true}: MakeCommand(func(e *InteractiveFilter[T]) error {
			return e.UserInputComponent.BeginningOfTheLine()
		}),
		Key{K: "e", Control: true}: MakeCommand(func(e *InteractiveFilter[T]) error {
			return e.UserInputComponent.EndOfTheLine()
		}),
		Key{K: "g", Control: true}: MakeCommand(func(e *InteractiveFilter[T]) error {
			e.parent.KillBuffer(e.ID)
			return nil
		}),

		Key{K: "<right>"}: MakeCommand(func(e *InteractiveFilter[T]) error {
			return e.UserInputComponent.CursorRight(1)
		}),
		Key{K: "<right>", Control: true}: MakeCommand(func(e *InteractiveFilter[T]) error {
			return e.UserInputComponent.NextWordStart()
		}),
		Key{K: "<left>"}: MakeCommand(func(e *InteractiveFilter[T]) error {
			return e.UserInputComponent.CursorLeft(1)
		}),
		Key{K: "<left>", Control: true}: MakeCommand(func(e *InteractiveFilter[T]) error {
			return e.UserInputComponent.PreviousWord()
		}),

		Key{K: "p", Control: true}: MakeCommand(func(e *InteractiveFilter[T]) error {
			e.List.PrevItem()
			return nil
		}),
		Key{K: "n", Control: true}: MakeCommand(func(e *InteractiveFilter[T]) error {
			e.List.NextItem()
			return nil
		}),
		Key{K: "<up>"}: MakeCommand(func(e *InteractiveFilter[T]) error {
			e.List.PrevItem()

			return nil
		}),
		Key{K: "<down>"}: MakeCommand(func(e *InteractiveFilter[T]) error {
			e.List.NextItem()
			return nil
		}),
		Key{K: "b", Control: true}: MakeCommand(func(e *InteractiveFilter[T]) error {
			return e.UserInputComponent.CursorLeft(1)
		}),
		Key{K: "<home>"}: MakeCommand(func(e *InteractiveFilter[T]) error {
			return e.UserInputComponent.BeginningOfTheLine()
		}),

		Key{K: "<enter>"}: MakeCommand(func(e *InteractiveFilter[T]) error {
			if len(e.List.Items) > 0 && len(e.List.Items) > e.List.Selection {
				return e.OpenSelection(e.parent, e.List.Items[e.List.Selection])
			}

			return nil
		}),
		Key{K: "<backspace>"}:                MakeCommand(func(e *InteractiveFilter[T]) error { return e.UserInputComponent.DeleteCharBackward() }),
		Key{K: "<backspace>", Control: true}: MakeCommand(func(e *InteractiveFilter[T]) error { return e.UserInputComponent.DeleteWordBackward() }),
		Key{K: "d", Control: true}:           MakeCommand(func(e *InteractiveFilter[T]) error { return e.UserInputComponent.DeleteCharForward() }),
		Key{K: "d", Alt: true}:               MakeCommand(func(e *InteractiveFilter[T]) error { return e.UserInputComponent.DeleteWordForward() }),
		Key{K: "<delete>"}:                   MakeCommand(func(e *InteractiveFilter[T]) error { return e.UserInputComponent.DeleteCharForward() }),
	}
}

func NewBufferSwitcher(parent *Context, cfg *Config) *InteractiveFilter[ScoredItem[Drawable]] {
	updateList := func(l *components.ListComponent[ScoredItem[Drawable]], input string) {
		for idx, item := range l.Items {
			l.Items[idx].Score = fuzzy.RankMatchNormalizedFold(input, fmt.Sprint(item.Item))
		}

		sortme(l.Items, func(t1 ScoredItem[Drawable], t2 ScoredItem[Drawable]) bool {
			return t1.Score > t2.Score
		})

	}
	openSelection := func(parent *Context, item ScoredItem[Drawable]) error {
		parent.KillBuffer(parent.ActiveBuffer().GetID())
		parent.MarkBufferAsActive(item.Item.GetID())

		return nil
	}
	initialList := func() []ScoredItem[Drawable] {
		var buffers []ScoredItem[Drawable]
		for _, v := range parent.Buffers {
			buffers = append(buffers, ScoredItem[Drawable]{Item: v})
		}

		return buffers
	}
	repr := func(s ScoredItem[Drawable]) string {
		return s.Item.String()
	}
	return NewInteractiveFilter[ScoredItem[Drawable]](
		parent,
		cfg,
		updateList,
		openSelection,
		repr,
		initialList,
	)

}

func NewThemeSwitcher(parent *Context, cfg *Config) *InteractiveFilter[ScoredItem[string]] {
	updateList := func(l *components.ListComponent[ScoredItem[string]], input string) {
		for idx, item := range l.Items {
			l.Items[idx].Score = fuzzy.RankMatchNormalizedFold(input, fmt.Sprint(item.Item))
		}

		sortme(l.Items, func(t1 ScoredItem[string], t2 ScoredItem[string]) bool {
			return t1.Score > t2.Score
		})

	}
	openSelection := func(parent *Context, item ScoredItem[string]) error {
		parent.Cfg.CurrentTheme = item.Item
		parent.KillBuffer(parent.ActiveBufferID())
		return nil
	}
	initialList := func() []ScoredItem[string] {
		var themes []ScoredItem[string]
		for _, v := range parent.Cfg.Themes {
			themes = append(themes, ScoredItem[string]{Item: v.Name})
		}

		return themes
	}
	repr := func(s ScoredItem[string]) string {
		return s.Item
	}
	return NewInteractiveFilter[ScoredItem[string]](
		parent,
		cfg,
		updateList,
		openSelection,
		repr,
		initialList,
	)

}

type GrepLocationItem struct {
	Filename string
	Text     string
	Line     int
	Col      int
}

type LocationItem struct {
	Filename string
}

func NewInteractiveFuzzyFile(parent *Context, cfg *Config, cwd string) *InteractiveFilter[ScoredItem[LocationItem]] {
	updateList := func(l *components.ListComponent[ScoredItem[LocationItem]], input string) {
		for idx, item := range l.Items {
			l.Items[idx].Score = fuzzy.RankMatchNormalizedFold(input, item.Item.Filename)
		}

		sortme(l.Items, func(t1 ScoredItem[LocationItem], t2 ScoredItem[LocationItem]) bool {
			return t1.Score > t2.Score
		})

	}
	openSelection := func(parent *Context, item ScoredItem[LocationItem]) error {
		err := SwitchOrOpenFileInTextBuffer(parent, parent.Cfg, path.Join(cwd, item.Item.Filename), nil)
		if err != nil {
			panic(err)
		}
		return nil
	}

	repr := func(g ScoredItem[LocationItem]) string {
		return fmt.Sprintf("%s", g.Item.Filename)
	}

	initialList := func() []ScoredItem[LocationItem] {
		var locationItems []ScoredItem[LocationItem]
		files := RipgrepFiles(cwd)
		for _, file := range files {
			locationItems = append(locationItems, ScoredItem[LocationItem]{Item: LocationItem{Filename: file}})
		}

		return locationItems

	}

	return NewInteractiveFilter[ScoredItem[LocationItem]](
		parent,
		cfg,
		updateList,
		openSelection,
		repr,
		initialList,
	)
}

func NewInteractiveFilePicker(parent *Context, cfg *Config, initialInput string) *InteractiveFilter[LocationItem] {
	updateList := func(l *components.ListComponent[LocationItem], input string) {
		matches, err := filepath.Glob(string(input) + "*")
		if err != nil {
			return
		}

		l.Items = nil

		for _, match := range matches {
			stat, err := os.Stat(match)
			if err == nil {
				isDir := stat.IsDir()
				_ = isDir
			}
			l.Items = append(l.Items, LocationItem{
				Filename: match,
			})
		}

		if l.Selection >= len(l.Items) {
			l.Selection = len(l.Items) - 1
		}

		if l.Selection < 0 {
			l.Selection = 0
		}

		return

	}
	openUserInput := func(parent *Context, userInput string) {
		parent.KillBuffer(parent.ActiveBufferID())
		err := SwitchOrOpenFileInTextBuffer(parent, parent.Cfg, userInput, nil)
		if err != nil {
			panic(err)
		}
	}
	openSelection := func(parent *Context, item LocationItem) error {
		parent.KillBuffer(parent.ActiveBufferID())
		err := SwitchOrOpenFileInTextBuffer(parent, parent.Cfg, item.Filename, nil)
		if err != nil {
			panic(err)
		}
		return nil
	}

	repr := func(g LocationItem) string {
		return fmt.Sprintf("%s", g.Filename)
	}

	tryComplete := func(f *InteractiveFilter[LocationItem]) error {
		input := f.UserInputComponent.UserInput

		matches, err := filepath.Glob(string(input) + "*")
		if err != nil {
			return nil
		}

		if len(matches) == 1 {
			stat, err := os.Stat(matches[0])
			if err == nil {
				if stat.IsDir() {
					matches[0] += "/"
				}
			}
			f.UserInputComponent.UserInput = []byte(matches[0])
			f.UserInputComponent.CursorRight(len(f.UserInputComponent.UserInput) - len(input))
		}
		return nil
	}

	ifb := NewInteractiveFilter[LocationItem](
		parent,
		cfg,
		updateList,
		openSelection,
		repr,
		nil,
	)

	ifb.keymaps[0][Key{K: "<enter>", Control: true}] = func(preditor *Context) error {
		input := preditor.ActiveBuffer().(*InteractiveFilter[LocationItem]).UserInputComponent.UserInput
		openUserInput(preditor, string(input))
		return nil
	}
	ifb.keymaps[0][Key{K: "<tab>"}] = MakeCommand(tryComplete)
	var absRoot string
	var err error
	if initialInput == "" {
		absRoot, _ = os.Getwd()
	} else {
		absRoot, err = filepath.Abs(initialInput)
		if err != nil {
			panic(err)
		}
	}
	ifb.UserInputComponent.SetNewUserInput([]byte(absRoot))

	return ifb
}
