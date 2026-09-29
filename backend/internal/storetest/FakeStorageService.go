package storetest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"

	"github.com/google/uuid"
	"github.com/kduong-dev/storage-service/pkg/storageservice"
)

// FakeStorageService is an in-memory storage service, holding each completed
// upload as a file.
type FakeStorageService struct {
	mutex   sync.Mutex
	uploads map[string]*bytes.Buffer
	keys    map[string]string
	Files   map[string][]byte
}

func NewFakeStorageService() *FakeStorageService {
	return &FakeStorageService{uploads: map[string]*bytes.Buffer{}, keys: map[string]string{}, Files: map[string][]byte{}}
}

var errNotFaked = errors.New("not faked")

func (fake *FakeStorageService) InitialiseUpload(ctx context.Context, input storageservice.InitialiseUploadInput) (*storageservice.UploadObject, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	id := uuid.NewString()
	fake.uploads[id] = &bytes.Buffer{}
	fake.keys[id] = input.Key
	return &storageservice.UploadObject{ID: id, Key: input.Key, ContentType: input.ContentType}, nil
}

func (fake *FakeStorageService) UploadPart(ctx context.Context, input storageservice.UploadPartInput) (*storageservice.UploadPartOutput, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	upload, ok := fake.uploads[input.UploadID]
	if !ok {
		return nil, storageservice.ErrUploadNotFound
	}
	size, err := io.Copy(upload, input.Body)
	return &storageservice.UploadPartOutput{PartNumber: input.PartNumber, Size: size}, err
}

func (fake *FakeStorageService) CompleteUpload(ctx context.Context, input storageservice.CompleteUploadInput) (*storageservice.FileObject, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	upload, ok := fake.uploads[input.UploadID]
	if !ok {
		return nil, storageservice.ErrUploadNotFound
	}
	delete(fake.uploads, input.UploadID)
	id := uuid.NewString()
	fake.Files[id] = upload.Bytes()
	return &storageservice.FileObject{ID: id, Key: fake.keys[input.UploadID], Size: int64(upload.Len())}, nil
}

func (fake *FakeStorageService) GetUploadObject(ctx context.Context, input storageservice.GetUploadObjectInput) (*storageservice.UploadObject, error) {
	return nil, errNotFaked
}

func (fake *FakeStorageService) AbortUpload(ctx context.Context, input storageservice.AbortUploadInput) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	delete(fake.uploads, input.UploadID)
	return nil
}

func (fake *FakeStorageService) DownloadFile(ctx context.Context, input storageservice.DownloadFileInput) (*storageservice.DownloadFileOutput, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	contents, ok := fake.Files[input.FileID]
	if !ok {
		return nil, storageservice.ErrFileNotFound
	}
	return &storageservice.DownloadFileOutput{Body: io.NopCloser(bytes.NewReader(contents))}, nil
}

func (fake *FakeStorageService) GetFileObject(ctx context.Context, input storageservice.GetFileObjectInput) (*storageservice.FileObject, error) {
	return nil, errNotFaked
}

func (fake *FakeStorageService) ListFileObjects(ctx context.Context, input storageservice.ListFileObjectsInput) (*storageservice.ListFileObjectsOutput, error) {
	return nil, errNotFaked
}

func (fake *FakeStorageService) MoveFile(ctx context.Context, input storageservice.MoveFileInput) (*storageservice.FileObject, error) {
	return nil, errNotFaked
}

func (fake *FakeStorageService) DeleteFile(ctx context.Context, input storageservice.DeleteFileInput) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if _, ok := fake.Files[input.FileID]; !ok {
		return storageservice.ErrFileNotFound
	}
	delete(fake.Files, input.FileID)
	return nil
}
