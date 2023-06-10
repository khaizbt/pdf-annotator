package pdf

type MetaDocument struct {
	Page        int64   `json:"page"`
	Width       float64 `json:"width"`
	Height      float64 `json:"height"`
	Orientation string  `json:"orientation"`
	Rotate      int     `json:"rotate"`
}
