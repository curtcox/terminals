package capability

import (
	"strings"
)

// PostBoard posts a non-pinned entry to a named board.
func (s *Service) PostBoard(board, text string) BoardItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.postBoardLocked(board, text, false)
}

// PinBoard pins a text item to the named board.
func (s *Service) PinBoard(board, text string) BoardItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.postBoardLocked(board, text, true)
}

func (s *Service) postBoardLocked(board, text string, pinned bool) BoardItem {
	item := BoardItem{
		ID:        s.nextIDLocked("pin"),
		Board:     defaultIfBlank(board, "default"),
		Pinned:    pinned,
		Text:      strings.TrimSpace(text),
		CreatedAt: s.now(),
	}
	s.boardItems = append(s.boardItems, item)
	action := "post"
	if pinned {
		action = "pin"
	}
	s.appendRecentLocked("board", item.ID+" "+action+" "+item.Text)
	return item
}

// ListBoard returns all items pinned to the given board.
func (s *Service) ListBoard(board string) []BoardItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	board = strings.TrimSpace(board)
	out := make([]BoardItem, 0, len(s.boardItems))
	for _, item := range s.boardItems {
		if board == "" || item.Board == board {
			out = append(out, item)
		}
	}
	return out
}
