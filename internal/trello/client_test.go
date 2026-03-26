package trello

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/members/me" {
			json.NewEncoder(w).Encode(map[string]string{"id": "user1", "username": "testuser"})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL
	err := c.ValidateCredentials()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateCredentialsUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	c := NewClient("badkey", "badtoken")
	c.BaseURL = server.URL
	err := c.ValidateCredentials()
	if err == nil {
		t.Fatal("expected error for unauthorized credentials")
	}
}

func TestFetchBoard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/boards/board1" {
			resp := map[string]interface{}{
				"id": "board1", "name": "Test Board", "url": "https://trello.com/b/board1",
				"lists": []map[string]interface{}{
					{"id": "list1", "name": "Backlog", "pos": 1.0},
					{"id": "list2", "name": "Done", "pos": 2.0},
				},
				"cards": []map[string]interface{}{
					{"id": "card1", "name": "Card One", "pos": 1.0, "idList": "list1", "url": "https://trello.com/c/card1", "labels": []interface{}{}},
					{"id": "card2", "name": "Card Two", "pos": 2.0, "idList": "list1", "url": "https://trello.com/c/card2", "labels": []interface{}{}},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL
	board, err := c.FetchBoard("board1")
	if err != nil { t.Fatalf("expected no error, got %v", err) }
	if board.Name != "Test Board" { t.Errorf("expected 'Test Board', got %q", board.Name) }
	if len(board.Lists) != 2 { t.Fatalf("expected 2 lists, got %d", len(board.Lists)) }
	if len(board.Lists[0].Cards) != 2 { t.Errorf("expected 2 cards in first list, got %d", len(board.Lists[0].Cards)) }
	if board.Lists[0].Cards[0].Name != "Card One" { t.Errorf("expected 'Card One', got %q", board.Lists[0].Cards[0].Name) }
}

func TestResolveBoard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/members/me/boards" {
			resp := []map[string]interface{}{
				{"id": "board1", "name": "My Board"}, {"id": "board2", "name": "Other Board"},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL
	id, err := c.ResolveBoard("My Board")
	if err != nil { t.Fatalf("expected no error, got %v", err) }
	if id != "board1" { t.Errorf("expected 'board1', got %q", id) }
}

func TestResolveBoardAmbiguous(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/members/me/boards" {
			resp := []map[string]interface{}{
				{"id": "board1", "name": "My Board"}, {"id": "board2", "name": "My Board"},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL
	_, err := c.ResolveBoard("My Board")
	if err == nil { t.Fatal("expected error for ambiguous board name") }
}

func TestMoveCardToList(t *testing.T) {
	var receivedBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/cards/card1" && r.Method == http.MethodPut {
			r.ParseForm()
			receivedBody = map[string]string{"idList": r.FormValue("idList"), "pos": r.FormValue("pos")}
			json.NewEncoder(w).Encode(map[string]string{"id": "card1"})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL
	err := c.MoveCardToList("card1", "list2", "top")
	if err != nil { t.Fatalf("expected no error, got %v", err) }
	if receivedBody["idList"] != "list2" { t.Errorf("expected idList 'list2', got %q", receivedBody["idList"]) }
}

func TestReorderCard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/cards/card1" && r.Method == http.MethodPut {
			json.NewEncoder(w).Encode(map[string]string{"id": "card1"})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL
	err := c.ReorderCard("card1", 12345.0)
	if err != nil { t.Fatalf("expected no error, got %v", err) }
}

func TestFetchCardComments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/cards/card1/actions" {
			filter := r.URL.Query().Get("filter")
			if filter != "commentCard" {
				t.Errorf("expected filter=commentCard, got %q", filter)
			}
			resp := []map[string]interface{}{
				{
					"id":   "action1",
					"date": "2026-03-20T10:30:00.000Z",
					"data": map[string]interface{}{
						"text": "This is a comment",
					},
					"memberCreator": map[string]interface{}{
						"id":       "member1",
						"fullName": "Craig Thomas",
						"initials": "CT",
						"username": "craigt",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL

	comments, err := c.FetchCardComments("card1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].Body != "This is a comment" {
		t.Errorf("expected body 'This is a comment', got %q", comments[0].Body)
	}
	if comments[0].Author.FullName != "Craig Thomas" {
		t.Errorf("expected author 'Craig Thomas', got %q", comments[0].Author.FullName)
	}
	if comments[0].Date.Year() != 2026 {
		t.Errorf("expected year 2026, got %d", comments[0].Date.Year())
	}
}

func TestFetchCardChecklists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/cards/card1/checklists" {
			resp := []map[string]interface{}{
				{
					"id":   "cl1",
					"name": "TODO",
					"checkItems": []map[string]interface{}{
						{"id": "ci1", "name": "First item", "state": "complete"},
						{"id": "ci2", "name": "Second item", "state": "incomplete"},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL

	checklists, err := c.FetchCardChecklists("card1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(checklists) != 1 {
		t.Fatalf("expected 1 checklist, got %d", len(checklists))
	}
	if checklists[0].Name != "TODO" {
		t.Errorf("expected name 'TODO', got %q", checklists[0].Name)
	}
	if len(checklists[0].Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(checklists[0].Items))
	}
	if !checklists[0].Items[0].Complete {
		t.Error("expected first item to be complete")
	}
	if checklists[0].Items[1].Complete {
		t.Error("expected second item to be incomplete")
	}
}

func TestCreateComment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/cards/card1/actions/comments" && r.Method == http.MethodPost {
			r.ParseForm()
			text := r.FormValue("text")
			if text != "Hello world" {
				t.Errorf("expected text 'Hello world', got %q", text)
			}
			resp := map[string]interface{}{
				"id":   "comment1",
				"date": "2026-03-26T10:30:00.000Z",
				"data": map[string]interface{}{
					"text": "Hello world",
				},
				"memberCreator": map[string]interface{}{
					"id":       "member1",
					"fullName": "Craig Thomas",
					"initials": "CT",
					"username": "craigt",
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL

	comment, err := c.CreateComment("card1", "Hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comment.ID != "comment1" {
		t.Errorf("expected ID 'comment1', got %q", comment.ID)
	}
	if comment.Body != "Hello world" {
		t.Errorf("expected body 'Hello world', got %q", comment.Body)
	}
	if comment.Author.FullName != "Craig Thomas" {
		t.Errorf("expected author 'Craig Thomas', got %q", comment.Author.FullName)
	}
	if !comment.Editable {
		t.Error("expected editable to be true for newly created comment")
	}
	if comment.Date.Year() != 2026 {
		t.Errorf("expected year 2026, got %d", comment.Date.Year())
	}
}

func TestUpdateComment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/actions/comment1" && r.Method == http.MethodPut {
			r.ParseForm()
			text := r.FormValue("text")
			if text != "Updated text" {
				t.Errorf("expected text 'Updated text', got %q", text)
			}
			resp := map[string]interface{}{
				"id":   "comment1",
				"date": "2026-03-26T10:35:00.000Z",
				"data": map[string]interface{}{
					"text": "Updated text",
				},
				"memberCreator": map[string]interface{}{
					"id":       "member1",
					"fullName": "Craig Thomas",
					"initials": "CT",
					"username": "craigt",
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL

	comment, err := c.UpdateComment("comment1", "Updated text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comment.ID != "comment1" {
		t.Errorf("expected ID 'comment1', got %q", comment.ID)
	}
	if comment.Body != "Updated text" {
		t.Errorf("expected body 'Updated text', got %q", comment.Body)
	}
}

func TestUpdateCommentNotSupported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/actions/comment1" && r.Method == http.MethodPut {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("update not supported"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL

	_, err := c.UpdateComment("comment1", "Updated text")
	if err == nil {
		t.Fatal("expected error for unsupported update")
	}
	if err != ErrUpdateNotSupported {
		t.Errorf("expected ErrUpdateNotSupported, got %v", err)
	}
}

func TestDeleteComment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/actions/comment1" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL

	err := c.DeleteComment("comment1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteCommentNotSupported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/actions/comment1" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("delete not supported"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL

	err := c.DeleteComment("comment1")
	if err == nil {
		t.Fatal("expected error for unsupported delete")
	}
	if err != ErrDeleteNotSupported {
		t.Errorf("expected ErrDeleteNotSupported, got %v", err)
	}
}

func TestGetBoardMembers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/boards/board1" {
			resp := map[string]interface{}{
				"members": []map[string]interface{}{
					{
						"id":       "member1",
						"fullName": "Craig Thomas",
						"initials": "CT",
						"username": "craigt",
					},
					{
						"id":       "member2",
						"fullName": "Jane Doe",
						"initials": "JD",
						"username": "janedoe",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := NewClient("testkey", "testtoken")
	c.BaseURL = server.URL

	members, err := c.GetBoardMembers("board1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
	if members[0].FullName != "Craig Thomas" {
		t.Errorf("expected first member 'Craig Thomas', got %q", members[0].FullName)
	}
	if members[1].Username != "janedoe" {
		t.Errorf("expected second member username 'janedoe', got %q", members[1].Username)
	}
}
