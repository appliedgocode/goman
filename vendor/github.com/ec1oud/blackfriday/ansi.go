//
// Blackfriday Markdown Processor
// Available at http://github.com/russross/blackfriday
//
// Copyright © 2017 Shawn Rutledge <s@ecloud.org>.
// Distributed under the Simplified BSD License.
// See README.md for details.
//

//
//
// ANSI terminal codes rendering backend
//
//

package blackfriday

import (
	"bytes"
	"strconv"
	"unicode"
	"html"
)

// Ansi renderer configuration options.
const (
	ANSI_SKIP_HTML                 = 1 << iota // skip preformatted HTML blocks
	ANSI_SKIP_STYLE                            // skip embedded <style> elements
	ANSI_SKIP_IMAGES                           // skip embedded images
	ANSI_SKIP_LINKS                            // skip all links
	ANSI_SAFELINK                              // only link to trusted protocols
	ANSI_USE_SMARTYPANTS                       // enable smart punctuation substitutions
	ANSI_SMARTYPANTS_FRACTIONS                 // enable smart fractions (with ANSI_USE_SMARTYPANTS)
	ANSI_SMARTYPANTS_DASHES                    // enable smart dashes (with ANSI_USE_SMARTYPANTS)
	ANSI_SMARTYPANTS_LATEX_DASHES              // enable LaTeX-style dashes (with ANSI_USE_SMARTYPANTS and ANSI_SMARTYPANTS_DASHES)
	ANSI_SMARTYPANTS_ANGLED_QUOTES             // enable angled double quotes (with ANSI_USE_SMARTYPANTS) for double quotes rendering
)

type AnsiRendererParameters struct {
	// Prepend this text to each relative URL.
	AbsolutePrefix string
	// Add this text to each footnote anchor, to ensure uniqueness.
	FootnoteAnchorPrefix string
	// Show this text inside the <a> tag for a footnote return link, if the
	// ANSI_FOOTNOTE_RETURN_LINKS flag is enabled. If blank, the string
	// <sup>[return]</sup> is used.
	FootnoteReturnLinkContents string
	// If set, add this text to the front of each Header ID, to ensure
	// uniqueness.
	HeaderIDPrefix string
	// If set, add this text to the back of each Header ID, to ensure uniqueness.
	HeaderIDSuffix string
}

// Ansi is a type that implements the Renderer interface for HTML output.
//
// Do not create this directly, instead use the AnsiRenderer function.
type Ansi struct {
	width    uint
	flags    int    // ANSI_* options
	closeTag string // how to end singleton tags: either " />" or ">"

	parameters AnsiRendererParameters

	// table of contents data
	headerCount  int
	currentLevel int
	toc          *bytes.Buffer

	// Track header IDs to prevent ID collision in a single generation.
	headerIDs map[string]int

	smartypants *smartypantsRenderer
}

// AnsiRenderer creates and configures an Ansi object, which
// satisfies the Renderer interface.
//
// flags is a set of ANSI_* options ORed together.
func AnsiRenderer(width int, flags int) Renderer {
	return AnsiRendererWithParameters(width, flags, AnsiRendererParameters{})
}

func AnsiRendererWithParameters(width int, flags int, renderParameters AnsiRendererParameters) Renderer {
	// configure the rendering engine
	closeTag := ""

	if renderParameters.FootnoteReturnLinkContents == "" {
		renderParameters.FootnoteReturnLinkContents = `<sup>[return]</sup>`
	}

	return &Ansi{
		width:      uint(width),
		flags:      flags,
		closeTag:   closeTag,
		parameters: renderParameters,

		headerCount:  0,
		currentLevel: 0,
		toc:          new(bytes.Buffer),

		headerIDs: make(map[string]int),

		smartypants: smartypants(flags),
	}
}

func (options *Ansi) GetFlags() int {
	return options.flags
}

// ----------------
// Utilities

// BreakLines word-wraps the given string within width in characters
// and returns a string slice with newline-terminated strings.
func BreakLines(s string, width uint) []string {
	init := make([]byte, 0, width + 1)
	buf := bytes.NewBuffer(init)
	ret := make([]string, 0, 4)

	var current uint
	var wordBuf bytes.Buffer

	for _, char := range s {
		if char == '\n' {
			char = ' '
		}
		if unicode.IsSpace(char) {
			if wordBuf.Len() > 0 {
				current += uint(1 + wordBuf.Len())
				buf.WriteRune(' ')
				wordBuf.WriteTo(buf)
				wordBuf.Reset()
			}
		} else {

			wordBuf.WriteRune(char)

			if current+uint(wordBuf.Len()) > width && uint(wordBuf.Len()) < width {
				buf.WriteRune('\n')
				ret = append(ret, buf.String())
				buf.Reset()
				current = 0
			}
		}
	}

	if wordBuf.Len() > 0 {
		buf.WriteRune(' ')
		wordBuf.WriteTo(buf)
	}
	if buf.Len() > 0 {
		ret = append(ret, buf.String())
	}

	return ret
}

func (options *Ansi) WriteWrapped(out *bytes.Buffer, text []byte, indent uint) {
	unesc := html.UnescapeString(string(text))
	wrapped := BreakLines(unesc, options.width - indent)
	for i, line := range wrapped {
		if (i > 0) {
			out.WriteString(`  `) // TODO the correct indent
		}
		out.WriteString(line)
	}
}

// ----------------
// output callbacks from Blackfriday

func (options *Ansi) TitleBlock(out *bytes.Buffer, text []byte) {
	text = bytes.TrimPrefix(text, []byte("% "))
	text = bytes.Replace(text, []byte("\n% "), []byte("\n"), -1)
	AnsiColor(out, '1', "33")
	out.Write(text)
	AnsiColor(out, '0', "0")
}

func (options *Ansi) Header(out *bytes.Buffer, text func() bool, level int, id string) {
	marker := out.Len()
	doubleSpace(out)

	AnsiColor(out, '1', "33")

	if !text() {
		out.Truncate(marker)
		return
	}

	AnsiColor(out, '0', "0")
}

func (options *Ansi) BlockHtml(out *bytes.Buffer, text []byte) {
	if options.flags&ANSI_SKIP_HTML != 0 {
		return
	}

	doubleSpace(out)
	out.WriteString(html.UnescapeString(string(text)))
	out.WriteByte('\n')
}

func (options *Ansi) HRule(out *bytes.Buffer) {
	doubleSpace(out)
	for i := uint(0); i < options.width; i++ {
		out.WriteRune('⎯')
	}
}

func (options *Ansi) BlockCode(out *bytes.Buffer, text []byte, lang string) {
	doubleSpace(out)
	out.WriteString(html.UnescapeString(string(text)))
}

func (options *Ansi) BlockQuote(out *bytes.Buffer, text []byte) {
	doubleSpace(out)
	unesc := html.UnescapeString(string(text))
	wrapped := BreakLines(unesc, options.width - uint(2))
	for _, line := range wrapped {
		out.WriteRune('⎸')
		out.WriteString(line)
	}
	out.WriteRune('\n')
}

func (options *Ansi) Table(out *bytes.Buffer, header []byte, body []byte, columnData []int) {
	doubleSpace(out)
	out.WriteString("<table>\n<thead>\n")
	out.Write(header)
	out.WriteString("</thead>\n\n<tbody>\n")
	out.Write(body)
	out.WriteString("</tbody>\n</table>\n")
}

func (options *Ansi) TableRow(out *bytes.Buffer, text []byte) {
	doubleSpace(out)
	out.WriteString("<tr>\n")
	out.Write(text)
	out.WriteString("\n</tr>\n")
}

func (options *Ansi) TableHeaderCell(out *bytes.Buffer, text []byte, align int) {
	doubleSpace(out)
	switch align {
	case TABLE_ALIGNMENT_LEFT:
		out.WriteString("<th align=\"left\">")
	case TABLE_ALIGNMENT_RIGHT:
		out.WriteString("<th align=\"right\">")
	case TABLE_ALIGNMENT_CENTER:
		out.WriteString("<th align=\"center\">")
	default:
		out.WriteString("<th>")
	}

	out.Write(text)
	out.WriteString("</th>")
}

func (options *Ansi) TableCell(out *bytes.Buffer, text []byte, align int) {
	doubleSpace(out)
	switch align {
	case TABLE_ALIGNMENT_LEFT:
		out.WriteString("<td align=\"left\">")
	case TABLE_ALIGNMENT_RIGHT:
		out.WriteString("<td align=\"right\">")
	case TABLE_ALIGNMENT_CENTER:
		out.WriteString("<td align=\"center\">")
	default:
		out.WriteString("<td>")
	}

	out.Write(text)
	out.WriteString("</td>")
}

func (options *Ansi) Footnotes(out *bytes.Buffer, text func() bool) {
	out.WriteString("⎯⎯⎯⎯⎯⎯⎯⎯\n")
	options.List(out, text, LIST_TYPE_ORDERED)
}

func (options *Ansi) FootnoteItem(out *bytes.Buffer, name, text []byte, flags int) {
	if flags&LIST_ITEM_CONTAINS_BLOCK != 0 || flags&LIST_ITEM_BEGINNING_OF_LIST != 0 {
		doubleSpace(out)
	}
	slug := slugify(name)
	out.WriteString(options.parameters.FootnoteAnchorPrefix)
	AnsiColor(out, '0', "33")
	out.WriteString(html.UnescapeString(string(slug)))
	out.WriteString(html.UnescapeString(string(text)))
	AnsiColor(out, '0', "0")
}

func (options *Ansi) List(out *bytes.Buffer, text func() bool, flags int) {
	marker := out.Len()
	doubleSpace(out)

	if !text() {
		out.Truncate(marker)
		return
	}
}

func AnsiColor(out *bytes.Buffer, n byte, c string) {
	out.WriteByte(0x1B)
	out.WriteByte('[')
	out.WriteByte(n)
	out.WriteByte(';')
	out.WriteString(c)
	out.WriteByte('m')
}

// YELLOW="\[\033[0;33m\]"

func (options *Ansi) ListItem(out *bytes.Buffer, text []byte, flags int) {

	indent := uint(3)
	//~ if (flags&LIST_ITEM_CONTAINS_BLOCK != 0 && flags&LIST_TYPE_DEFINITION == 0) ||
		//~ flags&LIST_ITEM_BEGINNING_OF_LIST != 0 {
		doubleSpace(out)
	//~ }
	if flags&LIST_TYPE_TERM != 0 {
		AnsiColor(out, '0', "33")
	} else if flags&LIST_TYPE_DEFINITION != 0 {
		out.WriteString("    ")
		indent = uint(4)
	}
	switch {
	case bytes.HasPrefix(text, []byte("[ ] ")):
		out.WriteString(` ☐`)
		text = text[3:]
	case bytes.HasPrefix(text, []byte("[x] ")) || bytes.HasPrefix(text, []byte("[X] ")):
		out.WriteString(` ✔`) // or could be ☒ or ☑
		text = text[3:]
	default:
		out.WriteString(` •`)
	}
	options.WriteWrapped(out, text, indent)
}

func (options *Ansi) Paragraph(out *bytes.Buffer, text func() bool) {
	marker := out.Len()
	doubleSpace(out)

	if !text() {
		out.Truncate(marker)
		return
	}
	out.WriteString("\n")
}

func (options *Ansi) AutoLink(out *bytes.Buffer, link []byte, kind int) {
	skipRanges := htmlEntity.FindAllIndex(link, -1)
	if options.flags&ANSI_SAFELINK != 0 && !isSafeLink(link) && kind != LINK_TYPE_EMAIL {
		// mark it but don't link it if it is not a safe link: no smartypants
		entityEscapeWithSkip(out, link, skipRanges)
		return
	}
	options.maybeWriteAbsolutePrefix(out, link)
	entityEscapeWithSkip(out, link, skipRanges)
}

func (options *Ansi) CodeSpan(out *bytes.Buffer, text []byte) {
	out.WriteString(html.UnescapeString(string(text)))
}

func (options *Ansi) DoubleEmphasis(out *bytes.Buffer, text []byte) {
	AnsiColor(out, '1', "35")
	out.WriteString(html.UnescapeString(string(text)))
	AnsiColor(out, '0', "0")
}

func (options *Ansi) Emphasis(out *bytes.Buffer, text []byte) {
	if len(text) == 0 {
		return
	}
	AnsiColor(out, '0', "35")
	out.WriteString(html.UnescapeString(string(text)))
	AnsiColor(out, '0', "0")
}

func (options *Ansi) maybeWriteAbsolutePrefix(out *bytes.Buffer, link []byte) {
	if options.parameters.AbsolutePrefix != "" && isRelativeLink(link) && link[0] != '.' {
		out.WriteString(options.parameters.AbsolutePrefix)
		if link[0] != '/' {
			out.WriteByte('/')
		}
	}
}

func (options *Ansi) Image(out *bytes.Buffer, link []byte, title []byte, alt []byte) {
}

func (options *Ansi) LineBreak(out *bytes.Buffer) {
}

func (options *Ansi) Link(out *bytes.Buffer, link []byte, title []byte, content []byte) {
	if options.flags&ANSI_SKIP_LINKS != 0 {
		out.WriteString(html.UnescapeString(string(content)))
		return
	}

	if options.flags&ANSI_SAFELINK != 0 && !isSafeLink(link) {
		out.WriteString(html.UnescapeString(string(content)))
		return
	}

	options.maybeWriteAbsolutePrefix(out, link)
	AnsiColor(out, '0', "34")
	out.WriteString(html.UnescapeString(string(content)))
	AnsiColor(out, '0', "0")
	return
}

func (options *Ansi) RawHtmlTag(out *bytes.Buffer, text []byte) {
	if options.flags&ANSI_SKIP_HTML != 0 {
		return
	}
	if options.flags&ANSI_SKIP_STYLE != 0 && isHtmlTag(text, "style") {
		return
	}
	if options.flags&ANSI_SKIP_LINKS != 0 && isHtmlTag(text, "a") {
		return
	}
	if options.flags&ANSI_SKIP_IMAGES != 0 && isHtmlTag(text, "img") {
		return
	}
	out.Write(text)
}

func (options *Ansi) TripleEmphasis(out *bytes.Buffer, text []byte) {
	AnsiColor(out, '1', "31")
	out.WriteString(html.UnescapeString(string(text)))
	AnsiColor(out, '0', "0")
}

func (options *Ansi) StrikeThrough(out *bytes.Buffer, text []byte) {
	AnsiColor(out, '9', "30")
	out.WriteString(html.UnescapeString(string(text)))
	AnsiColor(out, '0', "0")
}

func (options *Ansi) FootnoteRef(out *bytes.Buffer, ref []byte, id int) {
	out.Write(ref)
	AnsiColor(out, '1', "33")
	out.WriteString(strconv.Itoa(id))
	AnsiColor(out, '0', "0")
}

func (options *Ansi) Entity(out *bytes.Buffer, entity []byte) {
	out.WriteString(html.UnescapeString(string(entity)))
}

func (options *Ansi) NormalText(out *bytes.Buffer, text []byte) {
	if options.flags&ANSI_USE_SMARTYPANTS != 0 {
		options.Smartypants(out, text)
	} else {
		out.WriteString(html.UnescapeString(string(text)))
	}
}

func (options *Ansi) Smartypants(out *bytes.Buffer, text []byte) {
	smrt := smartypantsData{false, false}
	mark := 0
	for i := 0; i < len(text); i++ {
		if action := options.smartypants[text[i]]; action != nil {
			if i > mark {
				out.WriteString(html.UnescapeString(string(text[mark:i])))
			}

			previousChar := byte(0)
			if i > 0 {
				previousChar = text[i-1]
			}
			i += action(out, &smrt, previousChar, text[i:])
			mark = i + 1
		}
	}

	if mark < len(text) {
		out.WriteString(html.UnescapeString(string(text[mark:])))
	}
}

func (options *Ansi) DocumentHeader(out *bytes.Buffer) {
}

func (options *Ansi) DocumentFooter(out *bytes.Buffer) {
}

func (options *Ansi) TocHeaderWithAnchor(text []byte, level int, anchor string) {
}

func (options *Ansi) TocHeader(text []byte, level int) {
}

func (options *Ansi) TocFinalize() {
}

