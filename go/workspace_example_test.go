// SPDX-License-Identifier: EUPL-1.2

package tenant

import "context"

func ExampleWorkspaceContext_WithWorkspace() {
	holder := WorkspaceContext{Context: context.Background()}.WithWorkspace(&Workspace{UUID: "uuid-7"})
	holder.Workspace()
}

func ExampleWorkspaceContext_WithUser() {
	holder := WorkspaceContext{Context: context.Background()}.WithUser(&User{UUID: "user-9"})
	holder.User()
}

func ExampleWorkspaceContext_Workspace() {
	holder := WorkspaceContext{Context: WithWorkspace(context.Background(), &Workspace{UUID: "uuid-7"})}
	holder.Workspace()
}

func ExampleWorkspaceContext_User() {
	holder := WorkspaceContext{Context: WithUser(context.Background(), &User{UUID: "user-9"})}
	holder.User()
}

func ExampleWorkspaceFromCtx() {
	ctx := WithWorkspace(context.Background(), &Workspace{UUID: "uuid-7"})
	WorkspaceFromCtx(ctx)
}

func ExampleWithWorkspace() {
	ctx := WithWorkspace(context.Background(), &Workspace{UUID: "uuid-7"})
	WorkspaceFromCtx(ctx)
}
