# App


## Fields

| Field                                                 | Type                                                  | Required                                              | Description                                           |
| ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- |
| `ID`                                                  | `string`                                              | :heavy_check_mark:                                    | Unique identifier for the app                         |
| `Name`                                                | `string`                                              | :heavy_check_mark:                                    | Name of the app                                       |
| `StackID`                                             | `*string`                                             | :heavy_minus_sign:                                    | Optional existing stack ID claimed by this app        |
| `State`                                               | [*components.State](../../models/components/state.md) | :heavy_minus_sign:                                    | N/A                                                   |