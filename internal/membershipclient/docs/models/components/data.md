# Data


## Supported Types

### AuthenticationProviderResponseGoogleIDPConfig

```go
data := components.CreateDataAuthenticationProviderResponseGoogleIDPConfig(components.AuthenticationProviderResponseGoogleIDPConfig{/* values here */})
```

### AuthenticationProviderResponseMicrosoftIDPConfig

```go
data := components.CreateDataAuthenticationProviderResponseMicrosoftIDPConfig(components.AuthenticationProviderResponseMicrosoftIDPConfig{/* values here */})
```

### AuthenticationProviderResponseGithubIDPConfig

```go
data := components.CreateDataAuthenticationProviderResponseGithubIDPConfig(components.AuthenticationProviderResponseGithubIDPConfig{/* values here */})
```

### AuthenticationProviderResponseOIDCConfig

```go
data := components.CreateDataAuthenticationProviderResponseOIDCConfig(components.AuthenticationProviderResponseOIDCConfig{/* values here */})
```

## Union Discrimination

Use the `Type` field to determine which variant is active, then access the corresponding field:

```go
switch data.Type {
	case components.DataTypeAuthenticationProviderResponseGoogleIDPConfig:
		// data.AuthenticationProviderResponseGoogleIDPConfig is populated
	case components.DataTypeAuthenticationProviderResponseMicrosoftIDPConfig:
		// data.AuthenticationProviderResponseMicrosoftIDPConfig is populated
	case components.DataTypeAuthenticationProviderResponseGithubIDPConfig:
		// data.AuthenticationProviderResponseGithubIDPConfig is populated
	case components.DataTypeAuthenticationProviderResponseOIDCConfig:
		// data.AuthenticationProviderResponseOIDCConfig is populated
}
```
