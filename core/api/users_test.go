package api_test

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/cyril-fl/callback/core/api/services/auth"
	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/password"
	"github.com/cyril-fl/callback/core/services/process/setpassword"
	"github.com/cyril-fl/callback/core/services/testserver"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

var (
	multiInvitation = domain.ServerMode{Mode: "multi", AddUserMode: "invitation"}
	multiPublic     = domain.ServerMode{Mode: "multi", AddUserMode: "public"}
)

// session is the session cookie an answer sets, "" without one.
func session(response *http.Response) string {
	for _, cookie := range response.Cookies() {
		if cookie.Name == auth.Cookie {
			return cookie.Value
		}
	}
	return ""
}

// userID is the id of the account name.
func userID(t *testing.T, server *testserver.Server, name string) string {
	t.Helper()
	var id string
	if err := server.DB.NewSelect().Model((*domain.User)(nil)).Column("id").Where("name = ?", name).Scan(server.Ctx, &id); err != nil {
		t.Fatal(err)
	}
	return id
}

// The single mode logs in by the password alone, or with the account's name in any case; the multi mode needs the
// name, and a wrong name answers as a wrong password.
func TestLoginModes(t *testing.T) {
	login := func(server *testserver.Server, body map[string]any) int {
		t.Helper()
		var failure struct{ Error string }
		response := call(t, server, "", "POST", "/api/login", body, &failure)
		if response.StatusCode == http.StatusUnauthorized && failure.Error != "wrong name or password" {
			t.Errorf("login %v: %q", body, failure.Error)
		}
		return response.StatusCode
	}
	single := testserver.Serve(t)
	for _, c := range []struct {
		body map[string]any
		want int
	}{
		{map[string]any{"password": testserver.Password}, http.StatusOK},
		{map[string]any{"name": " ADMIN ", "password": testserver.Password}, http.StatusOK},
		{map[string]any{"name": "bob", "password": testserver.Password}, http.StatusUnauthorized},
		{map[string]any{"password": "wrong"}, http.StatusUnauthorized},
	} {
		if got := login(single, c.body); got != c.want {
			t.Errorf("single %v: %d, want %d", c.body, got, c.want)
		}
	}

	multi := testserver.ServeMode(t, multiInvitation)
	multi.Account(t, "bob")
	for _, c := range []struct {
		body map[string]any
		want int
	}{
		{map[string]any{"password": testserver.Password}, http.StatusUnauthorized},
		{map[string]any{"name": "Admin", "password": testserver.Password}, http.StatusOK},
		{map[string]any{"name": "bob", "password": testserver.Password}, http.StatusUnauthorized},
		{map[string]any{"name": "nobody", "password": testserver.Password}, http.StatusUnauthorized},
		{map[string]any{"name": "admin", "password": "wrong"}, http.StatusUnauthorized},
	} {
		if got := login(multi, c.body); got != c.want {
			t.Errorf("multi %v: %d, want %d", c.body, got, c.want)
		}
	}
	var me struct {
		User  string
		Admin bool
		Mode  string
	}
	if call(t, multi, multi.Session, "GET", "/api/me", nil, &me); me.User != "admin" || !me.Admin || me.Mode != "multi" {
		t.Errorf("me: %+v", me)
	}
}

// In the invitation mode, an invitation serves once and until it expires; the account it makes is no admin.
func TestSignupInvitation(t *testing.T) {
	server := testserver.ServeMode(t, multiInvitation)
	var mode domain.ServerMode
	if call(t, server, "", "GET", "/api/add-user", nil, &mode); mode != multiInvitation {
		t.Errorf("add-user: %+v", mode)
	}
	if response := call(t, server, "", "POST", "/api/signup", map[string]any{"name": "bob", "password": "long enough"}, nil); response.StatusCode != http.StatusForbidden {
		t.Errorf("no invitation: %d, want 403", response.StatusCode)
	}
	var made struct {
		Token  string
		Invite domain.Invite
	}
	call(t, server, server.Session, "POST", "/api/invites", nil, &made)
	var pending []domain.Invite
	if call(t, server, server.Session, "GET", "/api/invites", nil, &pending); made.Token == "" || len(pending) != 1 || pending[0].ID != made.Invite.ID {
		t.Fatalf("invite %+v, pending %+v", made, pending)
	}

	if response := call(t, server, "", "POST", "/api/signup", map[string]any{"name": "bob", "password": "short", "invite": made.Token}, nil); response.StatusCode != http.StatusBadRequest {
		t.Errorf("short password: %d, want 400", response.StatusCode)
	}
	if response := call(t, server, "", "POST", "/api/signup", map[string]any{"name": "ADMIN", "password": "long enough", "invite": made.Token}, nil); response.StatusCode != http.StatusBadRequest {
		t.Errorf("name taken: %d, want 400", response.StatusCode)
	}
	response := call(t, server, "", "POST", "/api/signup", map[string]any{"name": " bob ", "password": "long enough", "invite": made.Token}, nil)
	bob := session(response)
	var me struct {
		User  string
		Admin bool
	}
	if call(t, server, bob, "GET", "/api/me", nil, &me); response.StatusCode != http.StatusOK || me.User != "bob" || me.Admin {
		t.Errorf("signup: %d, me %+v", response.StatusCode, me)
	}
	if response := call(t, server, "", "POST", "/api/signup", map[string]any{"name": "carol", "password": "long enough", "invite": made.Token}, nil); response.StatusCode != http.StatusForbidden {
		t.Errorf("invitation used twice: %d, want 403", response.StatusCode)
	}
	if call(t, server, server.Session, "GET", "/api/invites", nil, &pending); len(pending) != 0 {
		t.Errorf("used invitation still pending: %+v", pending)
	}

	token := password.Token()
	past := time.Now().UTC().Add(-time.Hour)
	expired := domain.Invite{ID: uuid.NewString(), TokenHash: password.TokenHash(token), UserID: domain.UserID(server.Ctx), CreatedAt: past.Add(-domain.InviteLifetime), ExpiresAt: past}
	if _, err := server.DB.NewInsert().Model(&expired).Exec(server.Ctx); err != nil {
		t.Fatal(err)
	}
	if response := call(t, server, "", "POST", "/api/signup", map[string]any{"name": "carol", "password": "long enough", "invite": token}, nil); response.StatusCode != http.StatusForbidden {
		t.Errorf("expired invitation: %d, want 403", response.StatusCode)
	}

	call(t, server, server.Session, "POST", "/api/invites", nil, &made)
	if response := call(t, server, server.Session, "DELETE", "/api/invites/"+made.Invite.ID, nil, nil); response.StatusCode != http.StatusNoContent {
		t.Errorf("revoke: %d", response.StatusCode)
	}
	if response := call(t, server, "", "POST", "/api/signup", map[string]any{"name": "carol", "password": "long enough", "invite": made.Token}, nil); response.StatusCode != http.StatusForbidden {
		t.Errorf("revoked invitation: %d, want 403", response.StatusCode)
	}
}

// In the public mode anyone signs up, without an invitation.
func TestSignupPublic(t *testing.T) {
	server := testserver.ServeMode(t, multiPublic)
	response := call(t, server, "", "POST", "/api/signup", map[string]any{"name": "bob", "password": "long enough"}, nil)
	if response.StatusCode != http.StatusOK || session(response) == "" {
		t.Errorf("public signup: %d", response.StatusCode)
	}
}

// In the public mode an address creates a few accounts, then waits: a refused sign-up does not count.
func TestSignupPublicThrottled(t *testing.T) {
	server := testserver.ServeMode(t, multiPublic)
	if response := call(t, server, "", "POST", "/api/signup", map[string]any{"name": "", "password": "long enough"}, nil); response.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad name: %d, want 400", response.StatusCode)
	}
	for i := range 4 {
		name := "user" + strconv.Itoa(i)
		if response := call(t, server, "", "POST", "/api/signup", map[string]any{"name": name, "password": "long enough"}, nil); response.StatusCode != http.StatusOK {
			t.Fatalf("signup %s: %d, want 200", name, response.StatusCode)
		}
	}
	response := call(t, server, "", "POST", "/api/signup", map[string]any{"name": "user4", "password": "long enough"}, nil)
	if response.StatusCode != http.StatusTooManyRequests || response.Header.Get("Retry-After") == "" {
		t.Errorf("fifth account: %d (Retry-After %q), want 429", response.StatusCode, response.Header.Get("Retry-After"))
	}
	// The login of the address is not held back by its sign-ups.
	if response := call(t, server, "", "POST", "/api/login", map[string]any{"name": "user0", "password": "long enough"}, nil); response.StatusCode != http.StatusOK {
		t.Errorf("login after sign-ups: %d, want 200", response.StatusCode)
	}
}

// The routes of the accounts take an admin's session: another account or a token gets 403. In the single mode they
// do not exist, but GET /api/add-user.
func TestAccountRoutesGuards(t *testing.T) {
	server := testserver.ServeMode(t, multiInvitation)
	bob := server.Account(t, "bob")
	token := server.Token(t, false)
	routes := [][2]string{{"GET", "/api/users"}, {"DELETE", "/api/users/" + userID(t, server, "bob")}, {"POST", "/api/users/" + userID(t, server, "bob") + "/password"}, {"GET", "/api/invites"}, {"POST", "/api/invites"}, {"DELETE", "/api/invites/x"}}
	for _, route := range routes {
		for who, credential := range map[string]string{"bob": bob, "token": token} {
			if response := call(t, server, credential, route[0], route[1], nil, nil); response.StatusCode != http.StatusForbidden {
				t.Errorf("%s %s by %s: %d, want 403", route[0], route[1], who, response.StatusCode)
			}
		}
	}
	var users []domain.User
	if call(t, server, server.Session, "GET", "/api/users", nil, &users); len(users) != 2 || users[0].Name != "admin" || !users[0].Admin || users[1].Admin {
		t.Errorf("users: %+v", users)
	}

	single := testserver.Serve(t)
	for _, route := range append(routes, [2]string{"POST", "/api/signup"}) {
		if response := call(t, single, single.Session, route[0], route[1], map[string]any{}, nil); response.StatusCode != http.StatusNotFound {
			t.Errorf("single %s %s: %d, want 404", route[0], route[1], response.StatusCode)
		}
	}
	var mode domain.ServerMode
	if response := call(t, single, "", "GET", "/api/add-user", nil, &mode); response.StatusCode != http.StatusOK || mode.Mode != "single" {
		t.Errorf("single add-user: %d %+v", response.StatusCode, mode)
	}
}

// Deleting an account deletes all its data and nothing of the others'; an admin cannot delete themself.
func TestDeleteUser(t *testing.T) {
	server := testserver.ServeMode(t, multiInvitation)
	bob := server.Account(t, "bob")
	id := userID(t, server, "bob")
	for _, credential := range []string{server.Session, bob} {
		var created domain.BatchResult
		call(t, server, credential, "POST", "/api/applications", []map[string]any{{"company": "Acme", "location": "Paris"}, {"company": "Globex", "location": "Lyon"}}, &created)
		call(t, server, credential, "PATCH", "/api/applications", map[string]any{"ops": []map[string]any{{"ids": created.IDs, "set": map[string]any{"status": "interview", "followed_up": true}}}}, nil)
		call(t, server, credential, "DELETE", "/api/applications", map[string]any{"ids": created.IDs[1:]}, nil)
		call(t, server, credential, "POST", "/api/tokens", map[string]any{"name": "script"}, nil)
	}
	count := func(user string) map[string]int {
		t.Helper()
		counts := map[string]int{}
		for _, table := range []string{"applications", "deleted_applications", "tokens", "users"} {
			column := "user_id"
			if table == "users" {
				column = "id"
			}
			n, err := server.DB.NewSelect().Table(table).Where("? = ?", bun.Ident(column), user).Count(server.Ctx)
			if err != nil {
				t.Fatal(err)
			}
			counts[table] = int(n)
		}
		for _, table := range []string{"status_changes", "follow_ups"} {
			n, err := server.DB.NewSelect().Table(table).Where("application_id IN (SELECT id FROM applications WHERE user_id = ? UNION ALL SELECT id FROM deleted_applications WHERE user_id = ?)", user, user).Count(server.Ctx)
			if err != nil {
				t.Fatal(err)
			}
			counts[table] = int(n)
		}
		return counts
	}
	admin := domain.UserID(server.Ctx)
	before := count(admin)
	if bobs := count(id); bobs["applications"] != 1 || bobs["deleted_applications"] != 1 || bobs["status_changes"] == 0 || bobs["follow_ups"] == 0 || bobs["tokens"] == 0 {
		t.Fatalf("bob before: %v", bobs)
	}
	// The history of an application gone: bob's, once his applications are deleted.
	orphans := func() int {
		total := 0
		for _, table := range []string{"status_changes", "follow_ups"} {
			n, err := server.DB.NewSelect().Table(table).Where("application_id NOT IN (SELECT id FROM applications UNION ALL SELECT id FROM deleted_applications)").Count(server.Ctx)
			if err != nil {
				t.Fatal(err)
			}
			total += int(n)
		}
		return total
	}

	if response := call(t, server, server.Session, "DELETE", "/api/users/"+admin, nil, nil); response.StatusCode != http.StatusBadRequest {
		t.Errorf("delete self: %d, want 400", response.StatusCode)
	}
	if response := call(t, server, server.Session, "DELETE", "/api/users/nobody", nil, nil); response.StatusCode != http.StatusNotFound {
		t.Errorf("delete unknown: %d, want 404", response.StatusCode)
	}
	if response := call(t, server, server.Session, "DELETE", "/api/users/"+id, nil, nil); response.StatusCode != http.StatusNoContent {
		t.Fatalf("delete bob: %d", response.StatusCode)
	}
	for table, n := range count(id) {
		if n != 0 {
			t.Errorf("bob's %s left: %d", table, n)
		}
	}
	if after := count(admin); len(after) != len(before) || after["applications"] != before["applications"] || after["status_changes"] != before["status_changes"] || after["follow_ups"] != before["follow_ups"] || after["tokens"] != before["tokens"] {
		t.Errorf("admin's data: before %v, after %v", before, after)
	}
	if orphans() != 0 {
		t.Error("the history of bob's applications is left")
	}
	if response := call(t, server, bob, "GET", "/api/me", nil, nil); response.StatusCode != http.StatusUnauthorized {
		t.Errorf("bob's session after delete: %d, want 401", response.StatusCode)
	}
}

// An admin's reset gives a new password and logs the account's sessions out.
func TestResetPassword(t *testing.T) {
	server := testserver.ServeMode(t, multiInvitation)
	bob := server.Account(t, "bob")
	id := userID(t, server, "bob")
	var reset struct{ Password string }
	if response := call(t, server, server.Session, "POST", "/api/users/"+id+"/password", nil, &reset); response.StatusCode != http.StatusOK || len(reset.Password) < setpassword.MinLength {
		t.Fatalf("reset: %d %+v", response.StatusCode, reset)
	}
	if response := call(t, server, bob, "GET", "/api/me", nil, nil); response.StatusCode != http.StatusUnauthorized {
		t.Errorf("bob's session after reset: %d, want 401", response.StatusCode)
	}
	if response := call(t, server, "", "POST", "/api/login", map[string]any{"name": "bob", "password": reset.Password}, nil); response.StatusCode != http.StatusOK {
		t.Errorf("login with the new password: %d", response.StatusCode)
	}
	if response := call(t, server, server.Session, "POST", "/api/users/nobody/password", nil, nil); response.StatusCode != http.StatusNotFound {
		t.Errorf("reset unknown: %d, want 404", response.StatusCode)
	}
}
