package data

type User struct {
	Name string `json:"name"`
}

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PasswordEntry struct {
	ID       string `json:"id"`
	Service  string `json:"service"`
	Password string `json:"password"`
	Category string `json:"category"`
	Created  string `json:"created"`
}

type AppData struct {
	User       User            `json:"user"`
	Categories []Category      `json:"categories"`
	Entries    []PasswordEntry `json:"entries"`
}
