package controller

// GlobalCredentials almacena las credenciales globales
var SearchRulerActionCredentialsPool = &CredentialsStore{
	store: make(map[string]*Credentials),
}
