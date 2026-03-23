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
					{"id": "list1", "name": "Backlog", "pos": 1.0, "cards": []map[string]interface{}{
						{"id": "card1", "name": "Card One", "pos": 1.0, "idList": "list1", "url": "https://trello.com/c/card1", "labels": []interface{}{}},
						{"id": "card2", "name": "Card Two", "pos": 2.0, "idList": "list1", "url": "https://trello.com/c/card2", "labels": []interface{}{}},
					}},
					{"id": "list2", "name": "Done", "pos": 2.0, "cards": []map[string]interface{}{}},
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
