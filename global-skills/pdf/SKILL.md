---
name: pdf
description: >
  Generates branded PDF documents from markdown using pandoc + weasyprint.
  Use when the user says "/pdf", "generate PDF", "create PDF", "markdown to PDF",
  or asks about document generation workflows. Includes a complete starter CSS template.
---

# PDF Document Generation

Generate consistent, branded PDF documents from markdown using pandoc + weasyprint.

## Process

When triggered, follow these steps in order.

### 1. Verify Toolchain

Check that required tools are installed:

```bash
pandoc --version | head -1
weasyprint --version
```

If either is missing, tell the user:
```bash
brew install pandoc weasyprint
```

If the document has `.mmd` diagram sources, also check:
```bash
mmdc --version
```

If missing: `npm install -g @mermaid-js/mermaid-cli`

**Do not proceed until pandoc and weasyprint are confirmed.**

### 2. Discover Project Context

Scan the project to understand existing PDF infrastructure:

1. **Find existing stylesheets:** `find . -name "*.css" | grep -i print\|style\|pdf`
2. **Find existing PDFs:** `find . -name "*.pdf" -not -path "./.git/*"`
3. **Find mermaid sources:** `find . -name "*.mmd"`
4. **Check for docs structure:** `ls docs/ 2>/dev/null`

Report what you found. If the project already has a print stylesheet, ask the user whether to reuse it or create a new one.

### 3. Render Diagrams (if applicable)

If the markdown references `.mmd` diagrams or the user has mermaid source files:

```bash
# Single diagram
mmdc -i diagrams/my-diagram.mmd -o diagrams/my-diagram.png -t neutral -w 1200

# All diagrams in a directory
for f in diagrams/*.mmd; do
  mmdc -i "$f" -o "${f%.mmd}.png" -t neutral -w 1200
done
```

Flags:
- `-t neutral` — clean theme
- `-w 1200` — good width for letter-size PDF
- `-b transparent` — optional, for colored backgrounds

### 4. Create or Select Stylesheet

**If the project has an existing stylesheet:** use it (confirm with user).

**If creating a new one:** use the Starter CSS Template below. Customize these parts:
- `@top-center content` — set the document/project title
- `@bottom-right content` — set the version string (or remove the block)
- Color scheme — replace the 4 color variables throughout if the project has brand colors

Write the stylesheet next to the markdown source (e.g., `docs/my-feature/print-style.css`).

### 5. Generate PDF

Run pandoc with weasyprint:

```bash
cd <directory-containing-markdown>
pandoc DOCUMENT.md \
  -o Output-Name-v.1.0.pdf \
  --pdf-engine=weasyprint \
  --css=print-style.css \
  --metadata title="" \
  --pdf-engine-opt="--base-url=file://$(pwd)/"
```

**Required flags — never omit these:**
- `--css=` — always use a project stylesheet
- `--metadata title=""` — prevents pandoc from injecting a duplicate title
- `--base-url` — resolves relative image paths correctly

### 6. Verify Output

After generation:
1. Confirm the PDF file was created and has a reasonable file size
2. Report the output path to the user
3. If there were weasyprint warnings, explain which are harmless (see Troubleshooting)

---

## Starter CSS Template

Use this as a complete, working base for new documents. It handles page layout, typography, tables, code blocks, images, and smart page breaks out of the box.

```css
/* PDF print stylesheet — pandoc + weasyprint */
/* Customize: @page content strings and color values below */

/* ── Color reference ────────────────────────────────────────
   Primary text:     #2D2D2D  (dark gray)
   Accent:           #2563EB  (blue — h1 underline, blockquote border)
   Table headers:    #1E293B  (slate-900)
   Borders/dividers: #E2E8F0  (slate-200)
   Alternating rows: #F8FAFC  (slate-50)
   Blockquote bg:    #EFF6FF  (blue-50)
   Code bg:          #F1F5F9  (slate-100)
   ──────────────────────────────────────────────────────────── */

/* ── Page layout ─────────────────────────────────────────── */
@page {
  size: letter;
  margin: 25mm 20mm;
  @top-center {
    content: "Your Project — Document Title";          /* ← CHANGE THIS */
    font-size: 8pt;
    color: #999;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
  }
  @bottom-center {
    content: "Page " counter(page);
    font-size: 8pt;
    color: #999;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
  }
  /* Optional: version in bottom-right corner
  @bottom-right {
    content: "v1.0";
    font-size: 7pt;
    color: #bbb;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
  }
  */
}

/* First page: suppress header */
@page:first {
  @top-center { content: ""; }
}

/* ── Typography ──────────────────────────────────────────── */
body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
  font-size: 11pt;
  line-height: 1.5;
  color: #2D2D2D;
}

h1 {
  color: #2D2D2D;
  border-bottom: 3px solid #2563EB;
  padding-bottom: 8px;
  margin-top: 30px;
  font-size: 22pt;
  page-break-after: avoid;
}

/* h2 flows naturally: content fills pages instead of forcing every section
   onto a fresh page. For a deliberate break, add
   `<div class="page-break"></div>` in the markdown. */
h2 {
  color: #2D2D2D;
  border-bottom: 1px solid #E2E8F0;
  padding-bottom: 5px;
  margin-top: 24px;
  font-size: 16pt;
  page-break-before: auto;
  page-break-after: avoid;
  break-inside: avoid;
}

/* First h2 (subtitle) stays on title page */
h1 + h2 {
  page-break-before: avoid;
  border-bottom: none;
  font-size: 14pt;
  color: #555;
  margin-top: 0;
}

/* Manual page break utility: `<div class="page-break"></div>` */
.page-break {
  page-break-after: always;
  height: 0;
}

h3 {
  color: #333;
  margin-top: 18px;
  font-size: 13pt;
  page-break-after: avoid;
}

h4 {
  color: #555;
  font-size: 11pt;
  margin-top: 12px;
  page-break-after: avoid;
}

p {
  margin: 6px 0;
  orphans: 3;
  widows: 3;
}

/* ── Tables ──────────────────────────────────────────────── */
table {
  border-collapse: collapse;
  width: 100%;
  margin: 12px 0;
  font-size: 9.5pt;
  page-break-inside: auto;
}

th {
  background-color: #1E293B;
  color: white;
  padding: 8px 10px;
  text-align: left;
  font-weight: 600;
}

td {
  padding: 6px 10px;
  border-bottom: 1px solid #E2E8F0;
  vertical-align: top;
}

tr {
  page-break-inside: avoid;
}

tr:nth-child(even) td {
  background-color: #F8FAFC;
}

/* ── Code ────────────────────────────────────────────────── */
code {
  background-color: #F1F5F9;
  padding: 2px 5px;
  border-radius: 3px;
  font-size: 9pt;
  font-family: "SF Mono", Menlo, Monaco, "Courier New", monospace;
}

pre {
  background-color: #F8FAFC;
  border: 1px solid #E2E8F0;
  border-radius: 6px;
  padding: 12px;
  font-size: 8.5pt;
  line-height: 1.4;
  page-break-inside: avoid;
  overflow-wrap: break-word;
  white-space: pre-wrap;
}

pre code {
  background: none;
  padding: 0;
  font-size: inherit;
}

/* ── Blockquotes ─────────────────────────────────────────── */
blockquote {
  border-left: 4px solid #2563EB;
  margin: 16px 0;
  padding: 8px 16px;
  background: #EFF6FF;
  font-size: 9.5pt;
  page-break-inside: avoid;
}

blockquote p {
  margin: 4px 0;
}

/* ── Images ──────────────────────────────────────────────── */
img {
  max-width: 100%;
  max-height: 680px;
  height: auto;
  width: auto;
  display: block;
  margin: 8px auto;
  object-fit: contain;
  page-break-inside: avoid;
}

/* Keep heading + image together */
h2 + p > img,
h2 + img,
h3 + p > img {
  page-break-before: avoid;
}

/* Screenshot blockquotes: keep image + caption together */
blockquote:has(img) {
  page-break-inside: avoid;
  border-left: none;
  background: none;
  padding: 0;
  margin: 8px 0;
}

blockquote img {
  max-height: 480px;
  border: 1px solid #E2E8F0;
  border-radius: 4px;
}

/* ── Lists ───────────────────────────────────────────────── */
ul, ol {
  margin: 6px 0;
  padding-left: 22px;
}

li {
  margin: 3px 0;
}

li > ul, li > ol {
  margin: 2px 0;
}

/* ── Horizontal rules ────────────────────────────────────── */
hr {
  border: none;
  border-top: 1px solid #E2E8F0;
  margin: 16px 0;
}

/* ── Links ───────────────────────────────────────────────── */
a {
  color: #2563EB;
  text-decoration: none;
}

/* ── Strong / emphasis ───────────────────────────────────── */
strong {
  color: #2D2D2D;
}

/* ── Smart page breaks ───────────────────────────────────── */
h3 + table,
h3 + p,
h4 + ol,
h4 + ul,
h4 + p {
  page-break-before: avoid;
}

h1 + blockquote {
  page-break-before: avoid;
}

h3 + table {
  page-break-inside: avoid;
}

section {
  page-break-inside: auto;
}
```

---

## Markdown Conventions

Follow these rules when writing markdown for PDF output:

- **h1** (`#`) — document title only (one per document)
- **h2** (`##`) — major sections (flow naturally; insert `<div class="page-break"></div>` before one when you want a forced page break)
- **h3** (`###`) — subsections
- **`---`** — visual separator in source (renders as thin line)

### Front matter template

```markdown
# Document Title

## Subtitle or Description

**Prepared for:** Audience
**Date:** Month DD, YYYY
**Status:** Draft | Proposal | Approved

---
```

### Tables

Use standard markdown tables — no HTML or inline styles needed:

```markdown
| Column A | Column B |
|----------|----------|
| data     | data     |
```

### Images

Reference diagrams with relative paths from the markdown file's directory:

```markdown
![Caption](diagrams/my-diagram.png)
```

For screenshots with captions, use a blockquote:

```markdown
> ![Screenshot description](screenshots/my-screenshot.png)
> *Caption text here*
```

---

## File Naming

Keep generated PDFs next to their markdown source:

```
docs/<topic>/<Name>-v.<major>.<minor>.pdf
```

---

## Troubleshooting

| Issue | Fix |
|-------|-----|
| Images not showing in PDF | Add `--pdf-engine-opt="--base-url=file://$(pwd)/"` |
| Title rendered twice | Add `--metadata title=""` to suppress pandoc's auto-title |
| Warnings about `gap`, `overflow-x`, `user-select` | **Harmless** — pandoc injects default styles that weasyprint ignores |
| Tables breaking across pages | Add `page-break-inside: avoid` to the table CSS rule |
| Code blocks overflowing page width | Ensure CSS has `white-space: pre-wrap` and `overflow-wrap: break-word` on `pre` |
| Diagram not rendering | Check mmdc version (`mmdc --version`), try `-t neutral -w 1200` flags |
