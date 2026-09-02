package fixture

func setHome(t interface{ Setenv(string, string) }, dir string) {
	t.Setenv("HOME", dir)
}
