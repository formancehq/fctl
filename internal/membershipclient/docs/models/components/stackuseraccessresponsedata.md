# StackUserAccessResponseData


## Fields

| Field                                                   | Type                                                    | Required                                                | Description                                             |
| ------------------------------------------------------- | ------------------------------------------------------- | ------------------------------------------------------- | ------------------------------------------------------- |
| `StackID`                                               | *string*                                                | :heavy_check_mark:                                      | Stack ID                                                |
| `UserID`                                                | *string*                                                | :heavy_check_mark:                                      | User ID                                                 |
| `Email`                                                 | *string*                                                | :heavy_check_mark:                                      | User email                                              |
| `PolicyID`                                              | *int64*                                                 | :heavy_check_mark:                                      | Policy ID applied to the user for the stack             |
| `OrganizationPolicyID`                                  | **int64*                                                | :heavy_minus_sign:                                      | Policy ID applied to the user at the organization level |