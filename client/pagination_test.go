package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func int64Ptr(v int64) *int64 {
	return &v
}

func newPaginationTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return New("token").WithBaseURL(server.URL)
}

func requireQuery(t *testing.T, r *http.Request, key, want string) {
	t.Helper()

	if got := r.URL.Query().Get(key); got != want {
		t.Fatalf("query %q = %q, want %q", key, got, want)
	}
}

func TestListEnvironmentVariablesPageExposesPagination(t *testing.T) {
	client := newPaginationTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v8/projects/prj_123/env" {
			t.Fatalf("path = %q, want /v8/projects/prj_123/env", r.URL.Path)
		}
		requireQuery(t, r, "decrypt", "true")
		requireQuery(t, r, "teamId", "team_123")
		requireQuery(t, r, "limit", "25")
		requireQuery(t, r, "until", "222")
		fmt.Fprintln(w, `{
			"envs": [{"id":"env_1","key":"A"}],
			"pagination": {"count":1,"next":111,"prev":333}
		}`)
	})

	response, err := client.ListEnvironmentVariablesPage(context.Background(), ListEnvironmentVariablesRequest{
		ProjectID: "prj_123",
		TeamID:    "team_123",
		Limit:     25,
		Until:     int64Ptr(222),
	})
	if err != nil {
		t.Fatalf("ListEnvironmentVariablesPage() error = %v", err)
	}

	if len(response.EnvironmentVariables) != 1 {
		t.Fatalf("got %d env vars, want 1", len(response.EnvironmentVariables))
	}
	if response.EnvironmentVariables[0].TeamID != "team_123" {
		t.Fatalf("TeamID = %q, want team_123", response.EnvironmentVariables[0].TeamID)
	}
	if response.Pagination.Next == nil || *response.Pagination.Next != 111 {
		t.Fatalf("Next = %v, want 111", response.Pagination.Next)
	}
	if response.Pagination.Prev == nil || *response.Pagination.Prev != 333 {
		t.Fatalf("Prev = %v, want 333", response.Pagination.Prev)
	}
}

func TestGetEnvironmentVariablesPaginates(t *testing.T) {
	var seenUntil []string
	client := newPaginationTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		seenUntil = append(seenUntil, r.URL.Query().Get("until"))
		requireQuery(t, r, "decrypt", "true")
		requireQuery(t, r, "teamId", "team_123")
		requireQuery(t, r, "limit", "100")

		switch r.URL.Query().Get("until") {
		case "":
			fmt.Fprintln(w, `{
				"envs": [{"id":"env_1","key":"A"}],
				"pagination": {"count":1,"next":123}
			}`)
		case "123":
			fmt.Fprintln(w, `{
				"envs": [{"id":"env_2","key":"B"}],
				"pagination": {"count":1}
			}`)
		default:
			t.Fatalf("unexpected until %q", r.URL.Query().Get("until"))
		}
	})

	envs, err := client.GetEnvironmentVariables(context.Background(), "prj_123", "team_123")
	if err != nil {
		t.Fatalf("GetEnvironmentVariables() error = %v", err)
	}

	if got, want := len(envs), 2; got != want {
		t.Fatalf("got %d env vars, want %d", got, want)
	}
	if got, want := []string{envs[0].ID, envs[1].ID}, []string{"env_1", "env_2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("env IDs = %#v, want %#v", got, want)
	}
	if !reflect.DeepEqual(seenUntil, []string{"", "123"}) {
		t.Fatalf("seen until = %#v, want %#v", seenUntil, []string{"", "123"})
	}
}

func TestListSharedEnvironmentVariablesPaginates(t *testing.T) {
	client := newPaginationTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requireQuery(t, r, "teamId", "team_123")
		requireQuery(t, r, "limit", "100")

		switch r.URL.Query().Get("until") {
		case "":
			fmt.Fprintln(w, `{
				"data": [{"id":"env_1","key":"A"}],
				"pagination": {"count":1,"next":123}
			}`)
		case "123":
			fmt.Fprintln(w, `{
				"data": [{"id":"env_2","key":"B"}],
				"pagination": {"count":1}
			}`)
		default:
			t.Fatalf("unexpected until %q", r.URL.Query().Get("until"))
		}
	})

	envs, err := client.ListSharedEnvironmentVariables(context.Background(), "team_123")
	if err != nil {
		t.Fatalf("ListSharedEnvironmentVariables() error = %v", err)
	}

	if got, want := []string{envs[0].ID, envs[1].ID}, []string{"env_1", "env_2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("env IDs = %#v, want %#v", got, want)
	}
	if envs[0].TeamID != "team_123" || envs[1].TeamID != "team_123" {
		t.Fatalf("TeamID values = %#v, %#v; want team_123", envs[0].TeamID, envs[1].TeamID)
	}
}

func TestListProjectMembersPaginatesWithoutTeam(t *testing.T) {
	client := newPaginationTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/prj_123/members" {
			t.Fatalf("path = %q, want /v1/projects/prj_123/members", r.URL.Path)
		}
		requireQuery(t, r, "limit", "100")

		switch r.URL.Query().Get("until") {
		case "":
			fmt.Fprintln(w, `{
				"members": [{"uid":"user_1","role":"ADMIN"}],
				"pagination": {"count":1,"next":123}
			}`)
		case "123":
			fmt.Fprintln(w, `{
				"members": [{"uid":"user_2","role":"MEMBER"}],
				"pagination": {"count":1}
			}`)
		default:
			t.Fatalf("unexpected until %q", r.URL.Query().Get("until"))
		}
	})

	members, err := client.ListProjectMembers(context.Background(), GetProjectMembersRequest{
		ProjectID: "prj_123",
	})
	if err != nil {
		t.Fatalf("ListProjectMembers() error = %v", err)
	}

	if got, want := []string{members[0].UserID, members[1].UserID}, []string{"user_1", "user_2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("member IDs = %#v, want %#v", got, want)
	}
}

func TestGetTeamMemberPaginatesProjectAssignments(t *testing.T) {
	client := newPaginationTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/teams/team_123/members":
			requireQuery(t, r, "limit", "1")
			requireQuery(t, r, "filterByUserIds", "user_123")
			fmt.Fprintln(w, `{
				"members": [{
					"uid":"user_123",
					"email":"user@example.com",
					"role":"DEVELOPER",
					"confirmed":true
				}]
			}`)
		case "/v1/teams/team_123/members/user_123/projects":
			requireQuery(t, r, "limit", "100")
			switch r.URL.Query().Get("until") {
			case "":
				fmt.Fprintln(w, `{
					"projects": [{"projectId":"prj_1","role":"ADMIN"}],
					"pagination": {"count":1,"next":123}
				}`)
			case "123":
				fmt.Fprintln(w, `{
					"projects": [{"projectId":"prj_2","role":"MEMBER"}],
					"pagination": {"count":1}
				}`)
			default:
				t.Fatalf("unexpected until %q", r.URL.Query().Get("until"))
			}
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	})

	member, err := client.GetTeamMember(context.Background(), GetTeamMemberRequest{
		TeamID: "team_123",
		UserID: "user_123",
	})
	if err != nil {
		t.Fatalf("GetTeamMember() error = %v", err)
	}

	if got, want := []string{member.Projects[0].ProjectID, member.Projects[1].ProjectID}, []string{"prj_1", "prj_2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("project IDs = %#v, want %#v", got, want)
	}
}
