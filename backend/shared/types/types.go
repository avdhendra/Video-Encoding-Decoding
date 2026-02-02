// package types


// type PresignVideoUploadReq struct {
// 	Title         string `json:"title"`
// 	Description   string `json:"description"`
// 	VideoFilename string `json:"videoFilename"`
// 	VideoType     string `json:"videoType"`  // video/mp4
// 	ThumbFilename string `json:"thumbFilename"`
// 	ThumbType     string `json:"thumbType"`  // image/png, image/jpeg
// }


// type PresignVideoUploadResp struct {
// 	VideoID     string `json:"videoId"`
// 	VideoKey    string `json:"videoKey"`
// 	VideoPutURL string `json:"videoPutUrl"`

// 	ThumbKey    string `json:"thumbKey"`
// 	ThumbPutURL string `json:"thumbPutUrl"`
// }

// type PlaybackResp struct {
// 	VideoID            string   `json:"videoId"`
// 	JobID              *string  `json:"jobId,omitempty"`
// 	Status             string   `json:"status"`
// 	Progress           int      `json:"progress"`
// 	PlaybackReady      bool     `json:"playbackReady"`
// 	AvailableRenditions []string `json:"availableRenditions,omitempty"`
// 	MasterKey          *string  `json:"masterKey,omitempty"`
// 	MasterURL          string   `json:"masterUrl,omitempty"`
// }


package types

type PresignVideoUploadReq struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	VideoFilename string `json:"videoFilename"`
	VideoType     string `json:"videoType"` // video/mp4
	ThumbFilename string `json:"thumbFilename"`
	ThumbType     string `json:"thumbType"` // image/png, image/jpeg
}

type PresignVideoUploadResp struct {
	VideoID     string `json:"videoId"`
	VideoKey    string `json:"videoKey"`
	VideoPutURL string `json:"videoPutUrl"`

	ThumbKey    string `json:"thumbKey"`
	ThumbPutURL string `json:"thumbPutUrl"`
}

type CreateVideoJobReq struct {
	Pipeline string `json:"pipeline"` // "hls"
}

type PlaybackResp struct {
	VideoID             string   `json:"videoId"`
	JobID               *string  `json:"jobId,omitempty"`
	Status              string   `json:"status"`
	Progress            int      `json:"progress"`
	PlaybackReady        bool     `json:"playbackReady"`
	AvailableRenditions []string `json:"availableRenditions,omitempty"`
	MasterKey           *string  `json:"masterKey,omitempty"`
	MasterURL           string   `json:"masterUrl,omitempty"`
}


type TranscodeJobMessage struct {
	JobID    string `json:"jobId"`
	VideoID  string `json:"videoId"`
	InputKey string `json:"inputKey"`
	Pipeline string `json:"pipeline"` // "hls"
}