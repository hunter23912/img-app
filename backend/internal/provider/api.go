package provider

import (
	"context"
	"mime/multipart"
)

type ImageFile struct {
	File   multipart.File
	Header *multipart.FileHeader
}

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
	return CallEditImages(ctx, editURL, apiKey, input, []ImageFile{{File: imageFile, Header: imageHeader}}, maskFile, maskHeader, onEvent)
}

func CallEditImages(ctx context.Context, editURL, apiKey string, input ImageRequest, images []ImageFile, maskFile multipart.File, maskHeader *multipart.FileHeader, onEvent ImageEventHandler) (string, error) {
	return callRelayEditImagesWithContext(ctx, editURL, apiKey, input, images, maskFile, maskHeader, onEvent)
}
