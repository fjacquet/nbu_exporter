package models

// CatalogImagesCount captures only the pagination count from GET /catalog/images
// (queried with page[limit]=1 per filter combination to read the total cheaply).
//
// The JSON shape mirrors the existing Storages/Jobs response models: a top-level
// "meta" object containing a "pagination" object with a "count" field. This was
// verified against docs/veritas-11.2/catalog.yaml (the imageList schema's meta ->
// pagination -> paginationValues.count).
type CatalogImagesCount struct {
	Meta struct {
		Pagination struct {
			Count int64 `json:"count"`
		} `json:"pagination"`
	} `json:"meta"`
}
