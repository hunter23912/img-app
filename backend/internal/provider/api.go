package provider

import "mime/multipart"

func BuildImagesURL(endpoint, action string) (string, error) {
	return buildImagesURL(endpoint, action)
}

func CallGenerate(generateURL, apiKey string, input ImageRequest) (string, error) {
	return callRelayGenerate(generateURL, apiKey, input)
}

func CallEdit(
	editURL string,
	apiKey string,
	input ImageRequest,
	imageFile multipart.File,
	imageHeader *multipart.FileHeader,
	maskFile multipart.File,
	maskHeader *multipart.FileHeader,
) (string, error) {
	return callRelayEdit(editURL, apiKey, input, imageFile, imageHeader, maskFile, maskHeader)
}
