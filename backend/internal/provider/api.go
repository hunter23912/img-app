package provider

import (
	"context"
	"mime/multipart"
)

func BuildImagesURL(endpoint, action string) (string, error) {
	return buildImagesURL(endpoint, action)
}

func CallGenerate(ctx context.Context, generateURL, apiKey string, input ImageRequest, onEvent ImageEventHandler) (string, error) {
	return callRelayGenerateWithContext(ctx, generateURL, apiKey, input, onEvent)
}

func CallEdit(
	ctx context.Context,
	editURL string,
	apiKey string,
	input ImageRequest,
	imageFile multipart.File,
	imageHeader *multipart.FileHeader,
	maskFile multipart.File,
	maskHeader *multipart.FileHeader,
	onEvent ImageEventHandler,
) (string, error) {
	return callRelayEditWithContext(ctx, editURL, apiKey, input, imageFile, imageHeader, maskFile, maskHeader, onEvent)
}
