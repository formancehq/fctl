# RemoveDelegationFromScopeRequest


## Fields

| Field                                                         | Type                                                          | Required                                                      | Description                                                   | Example                                                       |
| ------------------------------------------------------------- | ------------------------------------------------------------- | ------------------------------------------------------------- | ------------------------------------------------------------- | ------------------------------------------------------------- |
| `ApplicationID`                                               | *string*                                                      | :heavy_check_mark:                                            | The unique identifier of the application (UUID format)        | 550e8400-e29b-41d4-a716-446655440000                          |
| `ScopeID`                                                     | *int64*                                                       | :heavy_check_mark:                                            | The unique identifier of the scope to operate on              |                                                               |
| `DelegatesToScopeID`                                          | *int64*                                                       | :heavy_check_mark:                                            | The ID of the organization scope that this scope delegates to | 49                                                            |