package sdjwttypes

var (
	TestAppAddr1 = "juno1jnd0n866f8ep2m2h2fnedry326qdcz77kkhqzy3j70dfl6u3uchs3qn3xp"
	TestAppAddr2 = "juno19kqlhsderagdyxrqa6md9yt3vpk4x3mnxxy5tm3j78nglhx2zd3qzvyjsr"
)

type InstantiateMsg struct {
	InitRegistrations  []InitRegistration `json:"init_registrations"`
	MaxPresentationLen uint               `json:"max_presentation_len"`
}

type InitRegistration struct {
	AppAdmin   string             `json:"app_admin"`
	AppAddress string             `json:"app_addr"`
	Routes     []RouteRequirement `json:"routes"`
}

type RouteVerificationRequirements struct {
	PresentationRequest Binary             `json:"presentation_request"`
	VerificationSource  VerificationSource `json:"verification_source"`
}

type Binary []byte

type TrustRegistry string

const (
	TrustRegistryCheqd TrustRegistry = "cheqd"
)

type VerificationSource struct {
	DataOrLocation Binary        `json:"data_or_location"`
	Source         TrustRegistry `json:"source,omitempty"`
}

type ExecuteMsg struct {
	Register   *Register   `json:"register,omitempty"`
	Verify     *Verify     `json:"verify,omitempty"`
	Update     *Update     `json:"update,omitempty"`
	Deregister *Deregister `json:"deregister,omitempty"`
}

type RouteRequirement struct {
	RouteId      uint64                        `json:"route_id"`
	Requirements RouteVerificationRequirements `json:"requirements"`
}

type Register struct {
	AppAddr       string             `json:"app_addr"`
	RouteCriteria []RouteRequirement `json:"route_criteria"`
}

type Verify struct {
	AppAddr      string `json:"app_addr,omitempty"`
	Presentation Binary `json:"presentation"`
	RouteID      uint64 `json:"route_id"`
}

type Update struct {
	AppAddr       string                         `json:"app_addr"`
	RouteCriteria *RouteVerificationRequirements `json:"route_criteria,omitempty"`
	RouteID       uint64                         `json:"route_id"`
}

type Deregister struct {
	AppAddr string `json:"app_addr"`
}

type QueryMsg struct {
	GetRoutes               *GetRoutes               `json:"get_routes,omitempty"`
	GetRouteRequirements    *GetRouteRequirements    `json:"get_route_requirements,omitempty"`
	GetRouteVerificationKey *GetRouteVerificationKey `json:"get_route_verification_key,omitempty"`
}

// In rust: Vec<RouteId>
type GetRoutesRes struct {
	Data []uint64 `json:"data"`
}
type GetRoutes struct {
	AppAddr string `json:"app_addr,omitempty"`
}

type GetRouteRequirements struct {
	AppAddr string `json:"app_addr,omitempty"`
	RouteID uint64 `json:"route_id,omitempty"`
}

// In rust: Option<String> of JwK
type GetRouteVerificationKeyRes struct {
	Data string `json:"data,omitempty"`
}
type GetRouteVerificationKey struct {
	AppAddr string `json:"app_addr,omitempty"`
	RouteID uint64 `json:"route_id,omitempty"`
}
