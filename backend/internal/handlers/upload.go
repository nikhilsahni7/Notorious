package handlers

import (
	"net/http"

	"notorious-backend/internal/services"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	uploadService *services.UploadService
}

func NewUploadHandler(uploadService *services.UploadService) *UploadHandler {
	return &UploadHandler{
		uploadService: uploadService,
	}
}

type InitUploadRequest struct {
	Filename string `json:"filename" binding:"required"`
	PartSize int64  `json:"part_size_mb"`
}

type InitUploadResponse struct {
	UploadID string `json:"upload_id"`
	Bucket   string `json:"bucket"`
	Key      string `json:"key"`
	PartSize int64  `json:"part_size_mb"`
}

type PresignPartRequest struct {
	UploadID   string `json:"upload_id" binding:"required"`
	Key        string `json:"key" binding:"required"`
	PartNumber int32  `json:"part_number" binding:"required"`
}

type CompleteUploadRequest struct {
	UploadID string                       `json:"upload_id" binding:"required"`
	Key      string                       `json:"key" binding:"required"`
	Parts    []CompletedUploadPartPayload `json:"parts" binding:"required"`
}

type CompletedUploadPartPayload struct {
	PartNumber int32  `json:"part_number" binding:"required"`
	ETag       string `json:"etag" binding:"required"`
}

func (h *UploadHandler) InitUpload(c *gin.Context) {
	var req InitUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.uploadService.InitMultipartUpload(req.Filename, req.PartSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *UploadHandler) PresignPart(c *gin.Context) {
	var req PresignPartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	url, err := h.uploadService.PresignPartUpload(req.UploadID, req.Key, req.PartNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

func (h *UploadHandler) CompleteUpload(c *gin.Context) {
	var req CompleteUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	completed := make([]types.CompletedPart, len(req.Parts))
	for i, part := range req.Parts {
		completed[i] = types.CompletedPart{
			ETag:       aws.String(part.ETag),
			PartNumber: aws.Int32(part.PartNumber),
		}
	}

	err := h.uploadService.CompleteMultipartUpload(req.UploadID, req.Key, completed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "upload completed"})
}

func (h *UploadHandler) AbortUpload(c *gin.Context) {
	var req CompleteUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.uploadService.AbortMultipartUpload(req.UploadID, req.Key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "upload aborted"})
}
