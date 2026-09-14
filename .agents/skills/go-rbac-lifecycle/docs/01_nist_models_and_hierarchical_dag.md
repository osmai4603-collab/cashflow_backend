# NIST RBAC Models and Hierarchical DAG Architecture in Go

This document details the formal mathematical modeling of the four standard NIST / ANSI INCITS 359-2012 RBAC levels and how they are implemented using idiomatic Go constructs.

---

## 1. Formal Models Defined by NIST

```text
       ┌────────────────────────┐
       │   4. Symmetric RBAC    │
       │   (Bidirectional Query)│
       └───────────▲────────────┘
                   │
       ┌───────────┴────────────┐
       │ 3. Constrained RBAC    │
       │ (SSD & DSD Separation) │
       └───────────▲────────────┘
                   │
       ┌───────────┴────────────┐
       │ 2. Hierarchical RBAC   │
       │ (Role Inheritance DAG) │
       └───────────▲────────────┘
                   │
       ┌───────────┴────────────┐
       │   1. Core / Flat RBAC  │
       │ (Users, Roles, Perms)  │
       └────────────────────────┘
```

### Level 1: Core (Flat) RBAC
- **Users ($U$)**: Entities (humans, system accounts, microservices) requesting access.
- **Roles ($R$)**: Functional titles grouping duties and responsibilities.
- **Permissions ($P = OP \times OBJ$)**: An operation (read, write, approve) paired with an object (invoices, users, orders).
- **User-to-Role Assignment ($UA \subseteq U \times R$)**: Many-to-many relationship mapping users to roles.
- **Permission-to-Role Assignment ($PA \subseteq P \times R$)**: Many-to-many relationship mapping permissions to roles.

### Level 2: Hierarchical RBAC
Introduces a partial order relation ($\succeq$) over $R$.
If $r_{senior} \succeq r_{junior}$, then:
$$P(r_{senior}) \supseteq P(r_{junior})$$
A user assigned to $r_{senior}$ automatically acquires all direct permissions of $r_{senior}$ plus all permissions inherited from $r_{junior}$.

### Level 3: Constrained RBAC (Separation of Duties)
Enforces security constraints preventing excessive authorization concentration:
- **Static Separation of Duties (SSD)**: A constraint specifying that no user can be assigned to both conflicting roles:
  $$\forall u \in U, \quad \{r_1, r_2\} \subseteq UA(u) \implies \text{SSD\_Conflict}(r_1, r_2) = \text{false}$$
- **Dynamic Separation of Duties (DSD)**: A constraint specifying that a user holding both roles cannot activate both concurrently within the same session/transaction:
  $$\forall s \in S(u), \quad \{r_1, r_2\} \subseteq ActiveRoles(s) \implies \text{DSD\_Conflict}(r_1, r_2) = \text{false}$$

### Level 4: Symmetric RBAC
Enforces system query symmetry:
- **Forward Query**: Given a user/role, list all authorized permissions.
- **Reverse Query**: Given a sensitive permission/resource, immediately enumerate all roles and subjects that hold that authority. Essential for automated security compliance and auditing.

---

## 2. Idiomatic Go Implementation of the Hierarchical DAG

In Go, a Directed Acyclic Graph (DAG) for role inheritance is modeled cleanly using standard hash maps:

```go
package rbac

type RoleDAG struct {
	// parentToChildren maps senior roles to junior roles they inherit from
	parentToChildren map[Role][]Role
}

func NewRoleDAG() *RoleDAG {
	return &RoleDAG{
		parentToChildren: make(map[Role][]Role),
	}
}

// AddInheritance registers that senior inherits from junior
func (dag *RoleDAG) AddInheritance(senior, junior Role) error {
	// Check for cycle before adding
	if dag.hasPath(junior, senior) {
		return fmt.Errorf("cyclic inheritance detected: %s -> %s would create a cycle", senior, junior)
	}
	dag.parentToChildren[senior] = append(dag.parentToChildren[senior], junior)
	return nil
}

// hasPath performs a Depth-First Search (DFS) to detect reachability
func (dag *RoleDAG) hasPath(start, target Role) bool {
	visited := make(map[Role]bool)
	var dfs func(current Role) bool
	dfs = func(current Role) bool {
		if current == target {
			return true
		}
		visited[current] = true
		for _, next := range dag.parentToChildren[current] {
			if !visited[next] {
				if dfs(next) {
					return true
				}
			}
		}
		return false
	}
	return dfs(start)
}

// ResolveAllInherited collects all ancestor and junior roles for a given set of initial roles
func (dag *RoleDAG) ResolveAllInherited(roles []Role) []Role {
	allRoles := make(map[Role]bool)
	var walk func(r Role)
	walk = func(r Role) {
		if allRoles[r] {
			return
		}
		allRoles[r] = true
		for _, child := range dag.parentToChildren[r] {
			walk(child)
		}
	}
	for _, r := range roles {
		walk(r)
	}

	result := make([]Role, 0, len(allRoles))
	for r := range allRoles {
		result = append(result, r)
	}
	return result
}
```

---

## 3. Reverse Query for Symmetric RBAC (Auditing)

To comply with Level 4 Symmetric RBAC, provide an inverted lookup index:

```go
// WhoHasPermission returns all subjects or roles that have access to the given permission
func (e *Engine) WhoCanPerform(targetPerm Permission) []Role {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var capableRoles []Role
	for role := range e.rolePerms {
		if e.HasPermission([]Role{role}, targetPerm) {
			capableRoles = append(capableRoles, role)
		}
	}
	return capableRoles
}
```
