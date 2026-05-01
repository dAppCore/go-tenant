// SPDX-License-Identifier: EUPL-1.2

package tenant

import (
	"context"

	"dappco.re/go"
)

func TestWorkspace_WorkspaceContext_WithWorkspace_Good(t *core.T) {
	holder := WorkspaceContext{Context: context.Background()}.WithWorkspace(testWorkspace())
	result := holder.Workspace()
	requireResultOK(t, result)
	core.AssertEqual(t, "uuid-7", result.Value.(*Workspace).UUID)
}

func TestWorkspace_WorkspaceContext_WithWorkspace_Bad(t *core.T) {
	holder := WorkspaceContext{Context: context.Background()}.WithWorkspace(nil)
	result := holder.Workspace()
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext, result.Value)
}

func TestWorkspace_WorkspaceContext_WithWorkspace_Ugly(t *core.T) {
	holder := WorkspaceContext{}.WithWorkspace(testWorkspace())
	result := holder.Workspace()
	requireResultOK(t, result)
	core.AssertEqual(t, "acme", result.Value.(*Workspace).Slug)
}

func TestWorkspace_WorkspaceContext_WithUser_Good(t *core.T) {
	holder := WorkspaceContext{Context: context.Background()}.WithUser(testUser())
	result := holder.User()
	requireResultOK(t, result)
	core.AssertEqual(t, "user-9", result.Value.(*User).UUID)
}

func TestWorkspace_WorkspaceContext_WithUser_Bad(t *core.T) {
	holder := WorkspaceContext{Context: context.Background()}.WithUser(nil)
	result := holder.User()
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoUserContext, result.Value)
}

func TestWorkspace_WorkspaceContext_WithUser_Ugly(t *core.T) {
	holder := WorkspaceContext{}.WithUser(testUser())
	result := holder.User()
	requireResultOK(t, result)
	core.AssertEqual(t, "ada@example.uk", result.Value.(*User).Email)
}

func TestWorkspace_WorkspaceContext_Workspace_Good(t *core.T) {
	holder := WorkspaceContext{Context: WithWorkspace(context.Background(), testWorkspace())}
	result := holder.Workspace()
	requireResultOK(t, result)
	core.AssertEqual(t, "uuid-7", result.Value.(*Workspace).UUID)
}

func TestWorkspace_WorkspaceContext_Workspace_Bad(t *core.T) {
	holder := WorkspaceContext{Context: context.Background()}
	result := holder.Workspace()
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext, result.Value)
}

func TestWorkspace_WorkspaceContext_Workspace_Ugly(t *core.T) {
	holder := WorkspaceContext{}
	result := holder.Workspace()
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext, result.Value)
}

func TestWorkspace_WorkspaceContext_User_Good(t *core.T) {
	holder := WorkspaceContext{Context: WithUser(context.Background(), testUser())}
	result := holder.User()
	requireResultOK(t, result)
	core.AssertEqual(t, "user-9", result.Value.(*User).UUID)
}

func TestWorkspace_WorkspaceContext_User_Bad(t *core.T) {
	holder := WorkspaceContext{Context: context.Background()}
	result := holder.User()
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoUserContext, result.Value)
}

func TestWorkspace_WorkspaceContext_User_Ugly(t *core.T) {
	holder := WorkspaceContext{}
	result := holder.User()
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoUserContext, result.Value)
}

func TestWorkspace_WorkspaceFromCtx_Good(t *core.T) {
	result := WorkspaceFromCtx(WithWorkspace(context.Background(), testWorkspace()))
	requireResultOK(t, result)
	core.AssertEqual(t, "uuid-7", result.Value.(*Workspace).UUID)
}

func TestWorkspace_WorkspaceFromCtx_Bad(t *core.T) {
	result := WorkspaceFromCtx(context.Background())
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext, result.Value)
}

func TestWorkspace_WorkspaceFromCtx_Ugly(t *core.T) {
	result := WorkspaceFromCtx(nil)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext, result.Value)
}

func TestWorkspace_WithWorkspace_Good(t *core.T) {
	ctx := WithWorkspace(context.Background(), testWorkspace())
	result := WorkspaceFromCtx(ctx)
	requireResultOK(t, result)
	core.AssertEqual(t, "acme", result.Value.(*Workspace).Slug)
}

func TestWorkspace_WithWorkspace_Bad(t *core.T) {
	ctx := WithWorkspace(context.Background(), nil)
	result := WorkspaceFromCtx(ctx)
	requireResultFail(t, result)
	core.AssertEqual(t, ErrNoWorkspaceContext, result.Value)
}

func TestWorkspace_WithWorkspace_Ugly(t *core.T) {
	ctx := WithWorkspace(nil, testWorkspace())
	result := WorkspaceFromCtx(ctx)
	requireResultOK(t, result)
	core.AssertEqual(t, "uuid-7", result.Value.(*Workspace).UUID)
}
