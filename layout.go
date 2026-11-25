package main

const defaultColumns = 5

type gridSize struct {
	Width  int
	Height int
}

func normalizeColumns(columns int) int {
	if columns < 1 {
		return defaultColumns
	}
	return columns
}

func buttonMetrics(availableWidth, columns int) (width, height, gap, margin int) {
	columns = normalizeColumns(columns)
	gap = 8
	margin = 8
	width = (availableWidth - 2*margin - gap*(columns-1)) / columns
	if width < 88 {
		width = 88
	}
	height = width / 2
	return width, height, gap, margin
}

func contentSize(buttonCount, columns int) gridSize {
	columns = normalizeColumns(columns)
	const (
		buttonWidth  = 136
		buttonHeight = 68
		gap          = 8
		margin       = 8
	)
	rows := buttonRows(buttonCount, columns)
	gaps := rows - 1
	if gaps < 0 {
		gaps = 0
	}
	return gridSize{
		Width:  margin*2 + columns*buttonWidth + (columns-1)*gap,
		Height: margin*2 + rows*buttonHeight + gaps*gap,
	}
}

func preferredWindowSize(buttonCount, columns int) gridSize {
	full := contentSize(buttonCount, columns)
	if buttonRows(buttonCount, columns) <= 4 {
		return full
	}
	capped := contentSize(columns*4, columns)
	return gridSize{Width: full.Width, Height: capped.Height}
}

func buttonRows(buttonCount, columns int) int {
	columns = normalizeColumns(columns)
	if buttonCount <= 0 {
		return 1
	}
	return (buttonCount + columns - 1) / columns
}
