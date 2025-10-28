# OrganizationExpanded


## Fields

| Field                                               | Type                                                | Required                                            | Description                                         |
| --------------------------------------------------- | --------------------------------------------------- | --------------------------------------------------- | --------------------------------------------------- |
| `Name`                                              | *string*                                            | :heavy_check_mark:                                  | Organization name                                   |
| `Domain`                                            | **string*                                           | :heavy_minus_sign:                                  | Organization domain                                 |
| `DefaultPolicyID`                                   | *int64*                                             | :heavy_check_mark:                                  | Default policy ID applied to new users              |
| `ID`                                                | *string*                                            | :heavy_check_mark:                                  | Organization ID                                     |
| `OwnerID`                                           | *string*                                            | :heavy_check_mark:                                  | Owner ID                                            |
| `AvailableStacks`                                   | **int64*                                            | :heavy_minus_sign:                                  | Number of available stacks                          |
| `AvailableSandboxes`                                | **int64*                                            | :heavy_minus_sign:                                  | Number of available sandboxes                       |
| `CreatedAt`                                         | [*time.Time](https://pkg.go.dev/time#Time)          | :heavy_minus_sign:                                  | N/A                                                 |
| `UpdatedAt`                                         | [*time.Time](https://pkg.go.dev/time#Time)          | :heavy_minus_sign:                                  | N/A                                                 |
| `TotalStacks`                                       | **int64*                                            | :heavy_minus_sign:                                  | N/A                                                 |
| `TotalUsers`                                        | **int64*                                            | :heavy_minus_sign:                                  | N/A                                                 |
| `Owner`                                             | [*components.User](../../models/components/user.md) | :heavy_minus_sign:                                  | N/A                                                 |