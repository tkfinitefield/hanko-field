package rbac

import "testing"

func TestHasCapabilityMatrix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		roles      []string
		capability Capability
		want       bool
	}{
		{
			name:       "admin has defined capability",
			roles:      []string{"admin"},
			capability: CapDashboardOverview,
			want:       true,
		},
		{
			name:       "admin denied for undefined capability",
			roles:      []string{"admin"},
			capability: Capability("made.up"),
			want:       false,
		},
		{
			name:       "ops can list orders",
			roles:      []string{"ops"},
			capability: CapOrdersList,
			want:       true,
		},
		{
			name:       "support cannot view system counters",
			roles:      []string{"support"},
			capability: CapSystemCounters,
			want:       false,
		},
		{
			name:       "marketing cannot view customers",
			roles:      []string{"marketing"},
			capability: CapCustomersView,
			want:       false,
		},
		{
			name:       "support can refund orders",
			roles:      []string{"support"},
			capability: CapOrderRefund,
			want:       true,
		},
		{
			name:       "ops has access to system tasks",
			roles:      []string{"ops"},
			capability: CapSystemTasks,
			want:       true,
		},
		{
			name:       "marketing gains reviews moderate",
			roles:      []string{"marketing"},
			capability: CapReviewsModerate,
			want:       true,
		},
		{
			name:       "combined roles inherit union of capabilities",
			roles:      []string{"support", "marketing"},
			capability: CapPromotionsManage,
			want:       true,
		},
		{
			name:       "unknown role grants nothing",
			roles:      []string{"unknown"},
			capability: CapOrdersList,
			want:       false,
		},
		{
			name:       "empty capability defaults to visible",
			roles:      []string{"support"},
			capability: Capability(""),
			want:       true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := HasCapability(tc.roles, tc.capability); got != tc.want {
				t.Fatalf("HasCapability(%v, %q) = %v, want %v", tc.roles, tc.capability, got, tc.want)
			}
		})
	}
}

func TestCapabilitiesForRoles(t *testing.T) {
	t.Parallel()

	caps := CapabilitiesForRoles([]string{"support"})
	if caps[CapOrderRefund] != true {
		t.Fatalf("support should have CapOrderRefund")
	}
	if caps[CapSystemCounters] {
		t.Fatalf("support must not have CapSystemCounters")
	}
}

func TestHasAnyRole(t *testing.T) {
	t.Parallel()

	if !HasAnyRole([]string{"support"}, Roles{RoleSupport}) {
		t.Fatal("support should satisfy role requirement")
	}
	if HasAnyRole([]string{"marketing"}, Roles{RoleSupport}) {
		t.Fatal("marketing should not satisfy support-only requirement")
	}
	if !HasAnyRole([]string{"marketing"}, Roles{RoleMarketing, RoleSupport}) {
		t.Fatal("marketing should satisfy marketing-or-support requirement")
	}
	if !HasAnyRole([]string{"unknown", "admin"}, Roles{RoleMarketing}) {
		t.Fatal("admin should satisfy requirement even when other roles unknown")
	}
}
