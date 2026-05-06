# Manifest


## Fields

| Field                                        | Type                                         | Required                                     | Description                                  |
| -------------------------------------------- | -------------------------------------------- | -------------------------------------------- | -------------------------------------------- |
| `ID`                                         | `string`                                     | :heavy_check_mark:                           | Unique identifier for the manifest           |
| `Name`                                       | `string`                                     | :heavy_check_mark:                           | Name of the manifest                         |
| `LatestVersion`                              | `int64`                                      | :heavy_check_mark:                           | Latest version number                        |
| `CreatedAt`                                  | [time.Time](https://pkg.go.dev/time#Time)    | :heavy_check_mark:                           | Timestamp when the manifest was created      |
| `UpdatedAt`                                  | [time.Time](https://pkg.go.dev/time#Time)    | :heavy_check_mark:                           | Timestamp when the manifest was last updated |