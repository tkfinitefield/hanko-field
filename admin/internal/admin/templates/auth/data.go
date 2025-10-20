package auth

// LoginPageData encapsulates rendering state for the admin login screen.
type LoginPageData struct {
	Email     string
	Message   string
	Error     string
	Remember  bool
	Next      string
	LoginPath string
	BasePath  string
	CSRFToken string
}
