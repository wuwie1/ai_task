package memory

import (
	"testing"
)

func TestParagraphChunker(t *testing.T) {
	chunker := &ParagraphChunker{
		config: DefaultChunkConfig(),
	}

	text := "这是第一段内容。这里有很多文字。\n\n这是第二段内容。继续写一些文字。\n\n这是第三段内容。最后一段。"

	chunks := chunker.Chunk(text, 50, 10)
	if len(chunks) == 0 {
		t.Error("Expected chunks, got none")
	}

	// 验证每个 chunk 都有正确的索引信息
	for i, chunk := range chunks {
		if chunk.ChunkIdx != i {
			t.Errorf("Expected ChunkIdx %d, got %d", i, chunk.ChunkIdx)
		}
		if chunk.TotalChunks != len(chunks) {
			t.Errorf("Expected TotalChunks %d, got %d", len(chunks), chunk.TotalChunks)
		}
		if chunk.Text == "" {
			t.Error("Chunk text should not be empty")
		}
	}
}

func TestSentenceChunker(t *testing.T) {
	chunker := &SentenceChunker{
		config: DefaultChunkConfig(),
	}

	text := "这是第一句话。这是第二句话！这是第三句话？这是第四句话。"

	chunks := chunker.Chunk(text, 30, 5)
	if len(chunks) == 0 {
		t.Error("Expected chunks, got none")
	}
}

func TestFixedChunker(t *testing.T) {
	chunker := &FixedChunker{
		config: DefaultChunkConfig(),
	}

	text := "这是一个很长的文本内容，需要被分割成多个固定大小的块。每个块都应该有重叠窗口，以确保上下文的连续性。"

	chunks := chunker.Chunk(text, 30, 5)
	if len(chunks) == 0 {
		t.Error("Expected chunks, got none")
	}

	// 验证重叠
	if len(chunks) > 1 {
		// 检查相邻块之间是否有重叠
		prevEnd := chunks[0].EndIdx
		for i := 1; i < len(chunks); i++ {
			if chunks[i].StartIdx >= prevEnd {
				t.Logf("Chunk %d starts at %d, previous chunk ended at %d (no overlap)", i, chunks[i].StartIdx, prevEnd)
			}
			prevEnd = chunks[i].EndIdx
		}
	}
}

func TestNewChunker(t *testing.T) {
	// 测试段落策略
	config := ChunkConfig{
		MaxSize:  100,
		Overlap:  10,
		MinSize:  20,
		Strategy: "paragraph",
	}
	chunker := NewChunker(config)
	if _, ok := chunker.(*ParagraphChunker); !ok {
		t.Error("Expected ParagraphChunker")
	}

	// 测试句子策略
	config.Strategy = "sentence"
	chunker = NewChunker(config)
	if _, ok := chunker.(*SentenceChunker); !ok {
		t.Error("Expected SentenceChunker")
	}

	// 测试固定长度策略
	config.Strategy = "fixed"
	chunker = NewChunker(config)
	if _, ok := chunker.(*FixedChunker); !ok {
		t.Error("Expected FixedChunker")
	}

	// 测试默认策略
	config.Strategy = ""
	chunker = NewChunker(config)
	if _, ok := chunker.(*ParagraphChunker); !ok {
		t.Error("Expected ParagraphChunker as default")
	}
}

func TestShortText(t *testing.T) {
	chunker := NewChunker(DefaultChunkConfig())
	text := "短文本"
	chunks := chunker.Chunk(text, 1000, 100)
	
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk for short text, got %d", len(chunks))
	}
	if chunks[0].Text != text {
		t.Errorf("Expected text %s, got %s", text, chunks[0].Text)
	}
}

