# UpsertAuthenticationProviderRequest


## Supported Types

### UpsertAuthenticationProviderRequestGoogleIDPConfig

```go
upsertAuthenticationProviderRequest := components.CreateUpsertAuthenticationProviderRequestUpsertAuthenticationProviderRequestGoogleIDPConfig(components.UpsertAuthenticationProviderRequestGoogleIDPConfig{/* values here */})
```

### UpsertAuthenticationProviderRequestMicrosoftIDPConfig

```go
upsertAuthenticationProviderRequest := components.CreateUpsertAuthenticationProviderRequestUpsertAuthenticationProviderRequestMicrosoftIDPConfig(components.UpsertAuthenticationProviderRequestMicrosoftIDPConfig{/* values here */})
```

### UpsertAuthenticationProviderRequestGithubIDPConfig

```go
upsertAuthenticationProviderRequest := components.CreateUpsertAuthenticationProviderRequestUpsertAuthenticationProviderRequestGithubIDPConfig(components.UpsertAuthenticationProviderRequestGithubIDPConfig{/* values here */})
```

### UpsertAuthenticationProviderRequestOIDCConfig

```go
upsertAuthenticationProviderRequest := components.CreateUpsertAuthenticationProviderRequestUpsertAuthenticationProviderRequestOIDCConfig(components.UpsertAuthenticationProviderRequestOIDCConfig{/* values here */})
```

## Union Discrimination

Use the `Type` field to determine which variant is active, then access the corresponding field:

```go
switch upsertAuthenticationProviderRequest.Type {
	case components.UpsertAuthenticationProviderRequestTypeUpsertAuthenticationProviderRequestGoogleIDPConfig:
		// upsertAuthenticationProviderRequest.UpsertAuthenticationProviderRequestGoogleIDPConfig is populated
	case components.UpsertAuthenticationProviderRequestTypeUpsertAuthenticationProviderRequestMicrosoftIDPConfig:
		// upsertAuthenticationProviderRequest.UpsertAuthenticationProviderRequestMicrosoftIDPConfig is populated
	case components.UpsertAuthenticationProviderRequestTypeUpsertAuthenticationProviderRequestGithubIDPConfig:
		// upsertAuthenticationProviderRequest.UpsertAuthenticationProviderRequestGithubIDPConfig is populated
	case components.UpsertAuthenticationProviderRequestTypeUpsertAuthenticationProviderRequestOIDCConfig:
		// upsertAuthenticationProviderRequest.UpsertAuthenticationProviderRequestOIDCConfig is populated
}
```
