# ReadVariablesResponseCursor

Cursor pagination envelope. `next` and `previous` are opaque tokens
produced by the server; pass them back as `?cursor=` to fetch the
adjacent page. `hasMore` is `true` when more results exist after the
current page. `pageSize` echoes the page size used for this page.



## Fields

| Field                                                        | Type                                                         | Required                                                     | Description                                                  |
| ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `PageSize`                                                   | `*int64`                                                     | :heavy_minus_sign:                                           | Number of items requested for this page                      |
| `HasMore`                                                    | `bool`                                                       | :heavy_check_mark:                                           | True when more results exist after this page                 |
| `Previous`                                                   | `*string`                                                    | :heavy_minus_sign:                                           | Opaque cursor token for the previous page (empty when none)  |
| `Next`                                                       | `*string`                                                    | :heavy_minus_sign:                                           | Opaque cursor token for the next page (empty when none)      |
| `Data`                                                       | [][components.Variable](../../models/components/variable.md) | :heavy_check_mark:                                           | N/A                                                          |