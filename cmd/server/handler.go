package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/suapapa/si-gnal/internal/player"
)

func (s *Server) handleGetPoems(c *gin.Context) {
	s.queueMu.Lock()
	defer s.queueMu.Unlock()

	c.JSON(http.StatusOK, s.poemQueue)
}

func (s *Server) handleGetPoemHead(c *gin.Context) {
	s.queueMu.Lock()
	if len(s.poemQueue) == 0 {
		s.queueMu.Unlock()
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Queue is empty"})
		return
	}
	job := s.poemQueue[0]
	s.queueMu.Unlock()

	play := c.Query("play")
	switch play {
	case "speaker":
		if err := s.playJobAsync(job, false); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, job)
	case "wav":
		if len(job.WavData) > 0 {
			c.Data(http.StatusOK, "audio/wav", job.WavData)
		} else {
			c.File(job.WavName)
		}
	default:
		c.JSON(http.StatusOK, job)
	}
}

func (s *Server) handleGetPoemPop(c *gin.Context) {
	s.queueMu.Lock()
	if len(s.poemQueue) == 0 {
		s.queueMu.Unlock()
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Queue is empty"})
		return
	}
	job := s.poemQueue[0]
	s.poemQueue = s.poemQueue[1:]
	s.queueMu.Unlock()

	play := c.Query("play")
	switch play {
	case "speaker":
		if err := s.playJobAsync(job, true); err != nil {
			// If play fails (e.g. already playing), we still popped it.
			// Maybe we should delete the file anyway?
			if len(job.WavData) == 0 {
				os.Remove(job.WavName)
			}
			c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "popped": job})
			return
		}
		c.JSON(http.StatusOK, job)
	case "wav":
		if len(job.WavData) > 0 {
			c.Data(http.StatusOK, "audio/wav", job.WavData)
		} else {
			c.File(job.WavName)
			os.Remove(job.WavName)
		}
	default:
		c.JSON(http.StatusOK, job)
		if len(job.WavData) == 0 {
			os.Remove(job.WavName)
		}
	}
}

func (s *Server) playJobAsync(job PlayJob, deleteAfter bool) error {
	s.playMu.Lock()
	if s.isPlaying {
		s.playMu.Unlock()
		return fmt.Errorf("already playing")
	}
	s.isPlaying = true
	var pCtx context.Context
	pCtx, s.playCancel = context.WithCancel(context.Background())
	s.currentPlayingFile = job.WavName
	s.playMu.Unlock()

	go func() {
		defer func() {
			s.playMu.Lock()
			s.isPlaying = false
			s.playCancel = nil
			if deleteAfter && len(job.WavData) == 0 {
				os.Remove(s.currentPlayingFile)
			}
			s.currentPlayingFile = ""
			s.playMu.Unlock()
		}()
		log.Printf("▶️ Playing %s...", job.WavName)
		var err error
		if len(job.WavData) > 0 {
			err = player.PlayWavFromBytes(pCtx, job.WavData)
		} else {
			err = player.PlayWav(pCtx, job.WavName)
		}
		if err != nil {
			log.Printf("failed to play %s: %v", job.WavName, err)
		}
	}()
	return nil
}

func (s *Server) handleStop(c *gin.Context) {
	s.playMu.Lock()
	defer s.playMu.Unlock()

	if !s.isPlaying {
		c.JSON(http.StatusOK, gin.H{"message": "Not playing"})
		return
	}

	if s.playCancel != nil {
		s.playCancel()
	}

	c.JSON(http.StatusOK, gin.H{"message": "Stopped"})
}
