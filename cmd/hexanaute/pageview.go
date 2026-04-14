package main

import (
	"image/color"
	"net/url"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/51TH-FireFox13/hexanaute/internal/engine"
)

// ── tappableBlock : wrapper cliquable autour de n'importe quel widget ────────
// MinSize retourne une largeur minimale de 32px pour éviter que le contenu
// non encore layouté ne force la fenêtre à s'élargir.

type tappableBlock struct {
	widget.BaseWidget
	content  fyne.CanvasObject
	onTapped func()
}

func newTappableBlock(content fyne.CanvasObject, onTapped func()) *tappableBlock {
	t := &tappableBlock{content: content, onTapped: onTapped}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tappableBlock) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.content)
}

// MinSize contraint la largeur à 32px (la largeur réelle est donnée par le
// parent lors du Layout), empêchant ainsi l'expansion de fenêtre.
func (t *tappableBlock) MinSize() fyne.Size {
	h := float32(16)
	if t.content != nil {
		h = t.content.MinSize().Height
	}
	return fyne.NewSize(32, h)
}

func (t *tappableBlock) Tapped(*fyne.PointEvent) {
	if t.onTapped != nil {
		t.onTapped()
	}
}

func (t *tappableBlock) TappedSecondary(*fyne.PointEvent) {}

// ── flowLayout : disposition horizontale avec retour à la ligne ───────────────
// Équivalent de CSS flex-wrap. Utilisé pour les barres de navigation.

type flowLayout struct{}

func (flowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	// On estime la hauteur en simulant le layout à 700px de large.
	const assumedW = float32(700)
	const gap = float32(4)
	x, rowH, totalH := float32(0), float32(0), float32(0)
	for i, o := range objects {
		s := o.MinSize()
		if x > 0 && x+s.Width > assumedW {
			x = 0
			totalH += rowH + gap
			rowH = 0
		}
		x += s.Width + gap
		if s.Height > rowH {
			rowH = s.Height
		}
		if i == len(objects)-1 {
			totalH += rowH
		}
	}
	if totalH == 0 {
		totalH = rowH
	}
	return fyne.NewSize(32, totalH)
}

func (flowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	const gap = float32(4)
	x, y, rowH := float32(0), float32(0), float32(0)
	for _, o := range objects {
		s := o.MinSize()
		if x > 0 && x+s.Width > size.Width {
			x = 0
			y += rowH + gap
			rowH = 0
		}
		o.Move(fyne.NewPos(x, y))
		o.Resize(s)
		x += s.Width + gap
		if s.Height > rowH {
			rowH = s.Height
		}
	}
}

func newFlowContainer(objects ...fyne.CanvasObject) *fyne.Container {
	return container.New(flowLayout{}, objects...)
}

// ── pageScrollWidget : scroll vertical à largeur contrainte ──────────────────
// container.NewVScroll propage la MinSize.Width de son contenu vers la fenêtre,
// ce qui provoque l'expansion à 5760px sur bureau multi-écrans.
// Ce widget plafonne la MinSize à 200×200 : Fyne donne ensuite la vraie largeur
// disponible lors du layout, le contenu se redimensionne correctement.

type pageScrollWidget struct {
	widget.BaseWidget
	inner *container.Scroll
}

func newPageScroll(content fyne.CanvasObject) *pageScrollWidget {
	s := &pageScrollWidget{}
	s.inner = container.NewVScroll(content)
	s.ExtendBaseWidget(s)
	return s
}

func (s *pageScrollWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.inner)
}

func (s *pageScrollWidget) MinSize() fyne.Size {
	return fyne.NewSize(200, 200)
}

func (s *pageScrollWidget) ScrollToTop() {
	s.inner.ScrollToTop()
}

// ── PageViewConfig ────────────────────────────────────────────────────────────

// PageViewConfig regroupe les callbacks nécessaires au rendu d'une page.
type PageViewConfig struct {
	OnLinkClick  func(url string)
	OnFormSubmit func(action, method string, data map[string]string)
	FetchImage   func(url string) ([]byte, error) // nil = pas d'images
	BaseURL      string
}

// ── buildPageView ─────────────────────────────────────────────────────────────

// buildPageView construit l'arbre Fyne à partir des blocs de rendu GUI.
// Les blocs de liste consécutifs qui sont des liens purs sont groupés en
// un flow container (rendu barre de navigation compacte qui wrappe).
func buildPageView(result *engine.GUIRenderResult, cfg PageViewConfig) fyne.CanvasObject {
	objects := make([]fyne.CanvasObject, 0, len(result.Blocks))

	i := 0
	for i < len(result.Blocks) {
		block := result.Blocks[i]

		// Détecter une séquence de BlockList/BlockParagraph quasi-exclusivement liens
		// → grouper en un flow container de boutons-liens
		if isPureLinkBlock(block) && cfg.OnLinkClick != nil {
			j := i
			var linkBtns []fyne.CanvasObject
			for j < len(result.Blocks) && isPureLinkBlock(result.Blocks[j]) {
				b := result.Blocks[j]
				for _, seg := range b.Segments {
					if seg.Link == "" {
						continue
					}
					linkURL := seg.Link
					text := cleanBullet(seg.Text)
					if text == "" {
						continue
					}
					btn := widget.NewButton(text, func() { cfg.OnLinkClick(linkURL) })
					btn.Importance = widget.LowImportance
					linkBtns = append(linkBtns, btn)
				}
				j++
			}
			if len(linkBtns) > 0 {
				objects = append(objects, container.NewPadded(newFlowContainer(linkBtns...)))
			}
			i = j
			continue
		}

		obj := renderBlock(block, cfg)
		if obj != nil {
			objects = append(objects, obj)
		}
		i++
	}

	if len(objects) == 0 {
		return widget.NewLabel("(page vide)")
	}
	return container.NewVBox(objects...)
}

// isPureLinkBlock retourne true si le bloc est un item de liste/paragraphe
// dont au moins 65% du texte (hors bullet) est du texte lié.
func isPureLinkBlock(block engine.GUIBlock) bool {
	if block.Type != engine.BlockList && block.Type != engine.BlockOrderedList &&
		block.Type != engine.BlockParagraph {
		return false
	}
	linkChars, totalChars := 0, 0
	for _, seg := range block.Segments {
		clean := len(cleanBullet(seg.Text))
		totalChars += clean
		if seg.Link != "" {
			linkChars += clean
		}
	}
	return totalChars > 0 && float64(linkChars)/float64(totalChars) >= 0.65
}

// cleanBullet supprime les préfixes de puce et espaces superflus.
func cleanBullet(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimLeft(s, "•◦▪\u00b7")
	s = strings.TrimLeft(s, "0123456789.")
	return strings.TrimSpace(s)
}

// ── renderBlock ───────────────────────────────────────────────────────────────

func renderBlock(block engine.GUIBlock, cfg PageViewConfig) fyne.CanvasObject {
	switch block.Type {
	case engine.BlockHeading1:
		return renderHeading(block, 24, true)
	case engine.BlockHeading2:
		return renderHeading(block, 20, true)
	case engine.BlockHeading3:
		return renderHeading(block, 17, false)
	case engine.BlockHeading4:
		return renderHeading(block, 15, false)
	case engine.BlockSep:
		sep := canvas.NewLine(color.NRGBA{R: 80, G: 80, B: 85, A: 255})
		sep.StrokeWidth = 1
		return container.NewStack(container.NewPadded(sep))
	case engine.BlockParagraph:
		return renderParagraph(block, cfg.OnLinkClick, 0)
	case engine.BlockList, engine.BlockOrderedList:
		return renderParagraph(block, cfg.OnLinkClick, block.Indent)
	case engine.BlockQuote:
		return renderBlockquote(block, cfg.OnLinkClick)
	case engine.BlockCodeBlock:
		return renderCodeBlock(block)
	case engine.BlockTable:
		return renderTableBlock(block)
	case engine.BlockImage:
		return renderImageBlock(block, cfg)
	case engine.BlockForm:
		return renderForm(block, cfg)
	case engine.BlockInputText, engine.BlockInputPassword,
		engine.BlockInputCheckbox, engine.BlockInputRadio,
		engine.BlockSelect, engine.BlockTextarea:
		return renderStandaloneField(block)
	case engine.BlockInputSubmit:
		label := block.InputValue
		if label == "" {
			label = "Envoyer"
		}
		return widget.NewButton(label, nil)
	default:
		return renderParagraph(block, cfg.OnLinkClick, 0)
	}
}

// ── Headings ─────────────────────────────────────────────────────────────────

func renderHeading(block engine.GUIBlock, size float32, bold bool) fyne.CanvasObject {
	text := segmentsToText(block.Segments)
	if text == "" {
		return nil
	}
	headingColor := color.NRGBA{R: 255, G: 180, B: 50, A: 255}
	if block.HasFGColor {
		headingColor = block.FGColor
	}
	heading := canvas.NewText(text, headingColor)
	heading.TextSize = size
	heading.TextStyle = fyne.TextStyle{Bold: bold}

	obj := fyne.CanvasObject(container.NewVBox(container.NewPadded(heading)))
	if block.HasBGColor {
		bg := canvas.NewRectangle(block.BGColor)
		obj = container.NewStack(bg, obj)
	}
	return obj
}

// ── Paragraphes / texte riche ────────────────────────────────────────────────

func renderParagraph(block engine.GUIBlock, onLinkClick func(string), indent int) fyne.CanvasObject {
	if len(block.Segments) == 0 {
		return nil
	}

	var firstLinkURL string
	richSegs := make([]widget.RichTextSegment, 0, len(block.Segments))
	for _, seg := range block.Segments {
		if seg.Text == "" {
			continue
		}
		if seg.Link != "" {
			if firstLinkURL == "" {
				firstLinkURL = seg.Link
			}
			// Lien : couleur primaire, sans annotation [n] (moins de bruit visuel)
			richSegs = append(richSegs, &widget.TextSegment{
				Text:  seg.Text,
				Style: widget.RichTextStyle{Inline: true, ColorName: "primary"},
			})
		} else {
			richSegs = append(richSegs, &widget.TextSegment{
				Text:  seg.Text,
				Style: segmentToRichStyle(seg),
			})
		}
	}
	if len(richSegs) == 0 {
		return nil
	}

	rt := widget.NewRichText(richSegs...)
	rt.Wrapping = fyne.TextWrapWord

	// Envelopper dans un tappableBlock si le paragraphe contient un lien.
	// MinSize() contraint la largeur à 32px → pas d'expansion de fenêtre.
	var obj fyne.CanvasObject = rt
	if firstLinkURL != "" && onLinkClick != nil {
		linkURL := firstLinkURL
		obj = newTappableBlock(rt, func() { onLinkClick(linkURL) })
	}

	if indent > 0 {
		obj = container.NewHBox(widget.NewLabel(strings.Repeat("  ", indent)), obj)
	}
	if block.Align == "center" {
		obj = container.NewCenter(obj)
	}
	if block.HasBGColor {
		bg := canvas.NewRectangle(block.BGColor)
		obj = container.NewStack(bg, container.NewPadded(obj))
	}
	return obj
}

func renderBlockquote(block engine.GUIBlock, onLinkClick func(string)) fyne.CanvasObject {
	text := segmentsToText(block.Segments)
	if text == "" {
		return nil
	}
	bar := canvas.NewRectangle(color.NRGBA{R: 255, G: 120, B: 0, A: 200})
	bar.SetMinSize(fyne.NewSize(3, 0))
	label := widget.NewRichTextFromMarkdown("*" + text + "*")
	label.Wrapping = fyne.TextWrapWord
	return container.NewHBox(bar, container.NewPadded(label))
}

func renderCodeBlock(block engine.GUIBlock) fyne.CanvasObject {
	text := segmentsToText(block.Segments)
	if text == "" {
		return nil
	}
	bg := canvas.NewRectangle(color.NRGBA{R: 30, G: 30, B: 35, A: 255})
	label := widget.NewLabel(text)
	label.TextStyle = fyne.TextStyle{Monospace: true}
	label.Wrapping = fyne.TextWrapWord
	return container.NewStack(bg, container.NewPadded(label))
}

func renderTableBlock(block engine.GUIBlock) fyne.CanvasObject {
	if len(block.Children) == 0 {
		return nil
	}
	rows := make([]fyne.CanvasObject, 0, len(block.Children))
	for i, row := range block.Children {
		text := segmentsToText(row.Segments)
		label := widget.NewLabel(text)
		if i == 0 {
			label.TextStyle = fyne.TextStyle{Bold: true}
		}
		label.Wrapping = fyne.TextWrapWord
		if i == 0 {
			bg := canvas.NewRectangle(color.NRGBA{R: 50, G: 50, B: 55, A: 255})
			rows = append(rows, container.NewStack(bg, container.NewPadded(label)))
		} else {
			rows = append(rows, container.NewPadded(label))
		}
	}
	bg := canvas.NewRectangle(color.NRGBA{R: 38, G: 38, B: 43, A: 255})
	return container.NewStack(bg, container.NewVBox(rows...))
}

// ── Images ───────────────────────────────────────────────────────────────────

func renderImageBlock(block engine.GUIBlock, cfg PageViewConfig) fyne.CanvasObject {
	alt := block.ImageAlt
	src := block.ImageSrc

	if src == "" {
		if alt == "" {
			return nil
		}
		lbl := widget.NewLabel("[Image: " + alt + "]")
		lbl.TextStyle = fyne.TextStyle{Italic: true}
		return lbl
	}

	resolvedSrc := resolveImageURL(cfg.BaseURL, src)

	placeholder := canvas.NewRectangle(color.NRGBA{R: 45, G: 45, B: 50, A: 255})
	placeholder.SetMinSize(fyne.NewSize(200, 120))
	altLabel := widget.NewLabel(alt)
	altLabel.TextStyle = fyne.TextStyle{Italic: true}
	imgContainer := container.NewStack(placeholder, container.NewCenter(altLabel))

	if cfg.FetchImage != nil {
		go func() {
			data, err := cfg.FetchImage(resolvedSrc)
			if err != nil || len(data) == 0 {
				altLabel.SetText("[Image: " + alt + "]")
				return
			}
			r := fyne.NewStaticResource(src, data)
			img := canvas.NewImageFromResource(r)
			img.FillMode = canvas.ImageFillContain
			img.SetMinSize(fyne.NewSize(300, 200))
			imgContainer.Objects = []fyne.CanvasObject{img}
			imgContainer.Refresh()
		}()
	}

	return imgContainer
}

// resolveImageURL résout une URL image relative par rapport à la base page.
func resolveImageURL(baseURL, imgSrc string) string {
	if imgSrc == "" || strings.HasPrefix(imgSrc, "data:") {
		return imgSrc
	}
	if strings.HasPrefix(imgSrc, "http://") || strings.HasPrefix(imgSrc, "https://") {
		return imgSrc
	}
	if strings.HasPrefix(imgSrc, "//") {
		if strings.HasPrefix(baseURL, "https:") {
			return "https:" + imgSrc
		}
		return "http:" + imgSrc
	}
	base, err := url.Parse(baseURL)
	if err != nil || base.Host == "" {
		return imgSrc
	}
	ref, err := url.Parse(imgSrc)
	if err != nil {
		return imgSrc
	}
	return base.ResolveReference(ref).String()
}

// ── Formulaires ──────────────────────────────────────────────────────────────

type formFieldRef struct {
	name   string
	kind   string // "entry", "check", "select", "textarea", "hidden"
	entry  *widget.Entry
	check  *widget.Check
	sel    *widget.Select
	hidden string
}

func renderForm(block engine.GUIBlock, cfg PageViewConfig) fyne.CanvasObject {
	var fields []formFieldRef
	objects := make([]fyne.CanvasObject, 0, len(block.Children)+1)

	topLine := canvas.NewRectangle(color.NRGBA{R: 255, G: 140, B: 0, A: 80})
	topLine.SetMinSize(fyne.NewSize(0, 1))
	objects = append(objects, topLine)

	for _, child := range block.Children {
		switch child.Type {
		case engine.BlockInputText:
			if child.InputType == "hidden" {
				fields = append(fields, formFieldRef{name: child.InputName, kind: "hidden", hidden: child.InputValue})
				continue
			}
			obj, ref := renderInputEntry(child)
			if ref != nil {
				fields = append(fields, *ref)
			}
			if obj != nil {
				objects = append(objects, obj)
			}

		case engine.BlockInputPassword:
			lbl, entry := labelledWidget(child.InputLabel, widget.NewPasswordEntry())
			entry.SetText(child.InputValue)
			entry.SetPlaceHolder(child.InputPlaceholder)
			if child.InputName != "" {
				e := entry
				fields = append(fields, formFieldRef{name: child.InputName, kind: "entry", entry: e})
			}
			objects = append(objects, lbl)

		case engine.BlockInputCheckbox:
			check := widget.NewCheck(child.InputLabel, nil)
			check.SetChecked(child.InputChecked)
			if child.InputDisabled {
				check.Disable()
			}
			if child.InputName != "" {
				c := check
				fields = append(fields, formFieldRef{name: child.InputName, kind: "check", check: c})
			}
			objects = append(objects, container.NewPadded(check))

		case engine.BlockInputRadio:
			check := widget.NewCheck("◉ "+child.InputLabel, nil)
			check.SetChecked(child.InputChecked)
			if child.InputName != "" {
				c := check
				fields = append(fields, formFieldRef{name: child.InputName, kind: "check", check: c})
			}
			objects = append(objects, container.NewPadded(check))

		case engine.BlockSelect:
			if len(child.SelectOptions) == 0 {
				continue
			}
			sel := widget.NewSelect(child.SelectOptions, nil)
			if child.InputValue != "" {
				for i, v := range child.SelectValues {
					if v == child.InputValue && i < len(child.SelectOptions) {
						sel.SetSelected(child.SelectOptions[i])
						break
					}
				}
			}
			if child.InputDisabled {
				sel.Disable()
			}
			if child.InputName != "" {
				s := sel
				sv := child.SelectValues
				so := child.SelectOptions
				fields = append(fields, formFieldRef{
					name:   child.InputName,
					kind:   "select",
					sel:    s,
					hidden: strings.Join(sv, "\x00") + "\xff" + strings.Join(so, "\x00"),
				})
			}
			var lbl fyne.CanvasObject
			if child.InputLabel != "" {
				lbl = container.NewBorder(nil, nil, widget.NewLabel(child.InputLabel+":"), nil, sel)
			} else {
				lbl = sel
			}
			objects = append(objects, container.NewPadded(lbl))

		case engine.BlockTextarea:
			entry := widget.NewMultiLineEntry()
			entry.SetText(child.InputValue)
			entry.SetPlaceHolder(child.InputPlaceholder)
			entry.SetMinRowsVisible(child.TextareaRows)
			if child.InputDisabled {
				entry.Disable()
			}
			if child.InputName != "" {
				e := entry
				fields = append(fields, formFieldRef{name: child.InputName, kind: "entry", entry: e})
			}
			var lbl fyne.CanvasObject
			if child.InputLabel != "" {
				lbl = container.NewBorder(widget.NewLabel(child.InputLabel+":"), nil, nil, nil, entry)
			} else {
				lbl = entry
			}
			objects = append(objects, container.NewPadded(lbl))

		case engine.BlockInputSubmit:
			label := child.InputValue
			if label == "" {
				label = "Envoyer"
			}
			if child.InputType == "reset" {
				objects = append(objects, container.NewPadded(widget.NewButton(label, nil)))
				continue
			}
			capturedFields := fields
			action := block.FormAction
			method := block.FormMethod
			btn := widget.NewButton(label, func() {
				if cfg.OnFormSubmit == nil {
					return
				}
				cfg.OnFormSubmit(action, method, collectFormData(capturedFields))
			})
			btn.Importance = widget.HighImportance
			objects = append(objects, container.NewPadded(btn))

		case engine.BlockParagraph:
			segs := child.Segments
			if len(segs) > 0 {
				rt := widget.NewRichText()
				rt.Wrapping = fyne.TextWrapWord
				var richSegs []widget.RichTextSegment
				for _, s := range segs {
					if s.Text != "" {
						richSegs = append(richSegs, &widget.TextSegment{
							Text: s.Text, Style: segmentToRichStyle(s),
						})
					}
				}
				rt.Segments = richSegs
				rt.Refresh()
				objects = append(objects, rt)
			}
		}
	}

	if len(objects) <= 1 {
		return nil
	}

	bg := canvas.NewRectangle(color.NRGBA{R: 35, G: 35, B: 42, A: 255})
	return container.NewStack(bg, container.NewPadded(container.NewVBox(objects...)))
}

func renderInputEntry(child engine.GUIBlock) (fyne.CanvasObject, *formFieldRef) {
	entry := widget.NewEntry()
	entry.SetText(child.InputValue)
	entry.SetPlaceHolder(child.InputPlaceholder)
	if child.InputDisabled {
		entry.Disable()
	}
	var ref *formFieldRef
	if child.InputName != "" {
		e := entry
		ref = &formFieldRef{name: child.InputName, kind: "entry", entry: e}
	}
	obj, _ := labelledWidget(child.InputLabel, entry)
	return obj, ref
}

func labelledWidget(label string, w fyne.CanvasObject) (fyne.CanvasObject, *widget.Entry) {
	entry, _ := w.(*widget.Entry)
	if label != "" {
		lbl := widget.NewLabel(label + ":")
		lbl.TextStyle = fyne.TextStyle{Bold: true}
		return container.NewBorder(nil, nil, lbl, nil, w), entry
	}
	return container.NewPadded(w), entry
}

func collectFormData(fields []formFieldRef) map[string]string {
	data := make(map[string]string, len(fields))
	for _, f := range fields {
		if f.name == "" {
			continue
		}
		switch f.kind {
		case "entry":
			if f.entry != nil {
				data[f.name] = f.entry.Text
			}
		case "check":
			if f.check != nil && f.check.Checked {
				data[f.name] = "on"
			}
		case "select":
			if f.sel != nil && f.sel.Selected != "" {
				val := f.sel.Selected
				if f.hidden != "" {
					parts := strings.SplitN(f.hidden, "\xff", 2)
					if len(parts) == 2 {
						vals := strings.Split(parts[0], "\x00")
						opts := strings.Split(parts[1], "\x00")
						for i, opt := range opts {
							if opt == val && i < len(vals) {
								val = vals[i]
								break
							}
						}
					}
				}
				data[f.name] = val
			}
		case "hidden":
			data[f.name] = f.hidden
		}
	}
	return data
}

func renderStandaloneField(block engine.GUIBlock) fyne.CanvasObject {
	switch block.Type {
	case engine.BlockInputText:
		e := widget.NewEntry()
		e.SetPlaceHolder(block.InputPlaceholder)
		e.SetText(block.InputValue)
		return container.NewPadded(e)
	case engine.BlockInputPassword:
		e := widget.NewPasswordEntry()
		e.SetPlaceHolder(block.InputPlaceholder)
		return container.NewPadded(e)
	case engine.BlockInputCheckbox:
		return container.NewPadded(widget.NewCheck(block.InputLabel, nil))
	case engine.BlockSelect:
		return container.NewPadded(widget.NewSelect(block.SelectOptions, nil))
	case engine.BlockTextarea:
		e := widget.NewMultiLineEntry()
		e.SetPlaceHolder(block.InputPlaceholder)
		e.SetMinRowsVisible(block.TextareaRows)
		return container.NewPadded(e)
	}
	return nil
}

// ── Styles ───────────────────────────────────────────────────────────────────

func segmentToRichStyle(seg engine.RichSegment) widget.RichTextStyle {
	style := widget.RichTextStyle{Inline: true}
	switch seg.Style {
	case engine.StyleBold:
		style.TextStyle.Bold = true
	case engine.StyleItalic:
		style.TextStyle.Italic = true
	case engine.StyleCode:
		style.TextStyle.Monospace = true
	case engine.StyleSmall:
		style.TextStyle.Italic = true
	case engine.StyleTableHeader:
		style.TextStyle.Bold = true
	case engine.StyleLink:
		style.ColorName = "primary"
	case engine.StyleImage:
		style.TextStyle.Italic = true
	}
	if seg.Bold {
		style.TextStyle.Bold = true
	}
	if seg.Italic {
		style.TextStyle.Italic = true
	}
	if seg.Mono {
		style.TextStyle.Monospace = true
	}
	return style
}

func segmentsToText(segs []engine.RichSegment) string {
	var buf strings.Builder
	for _, seg := range segs {
		buf.WriteString(seg.Text)
	}
	return strings.TrimSpace(buf.String())
}
