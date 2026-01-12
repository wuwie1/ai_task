package memory

import (
	"regexp"
	"strings"
)

// ChunkStrategy 分块策略接口
type ChunkStrategy interface {
	// Chunk 将文本分块
	// text: 待分块的文本
	// maxSize: 最大块大小（字符数）
	// overlap: 重叠窗口大小（字符数）
	// 返回: chunks 列表，每个 chunk 包含文本和索引信息
	Chunk(text string, maxSize, overlap int) []Chunk
}

// Chunk 表示一个文本块
type Chunk struct {
	Text      string // 块文本
	StartIdx  int    // 在原文本中的起始位置
	EndIdx    int    // 在原文本中的结束位置
	ChunkIdx  int    // 块索引（从0开始）
	TotalChunks int  // 总块数
}

// ChunkConfig 分块配置
type ChunkConfig struct {
	MaxSize    int    // 最大块大小（字符数），默认 1000
	Overlap    int    // 重叠窗口大小（字符数），默认 100
	MinSize    int    // 最小块大小（字符数），默认 200
	Strategy   string // 分块策略: "paragraph", "sentence", "fixed"，默认 "paragraph"
}

// DefaultChunkConfig 返回默认配置
func DefaultChunkConfig() ChunkConfig {
	return ChunkConfig{
		MaxSize:  1000,
		Overlap:  100,
		MinSize:  200,
		Strategy: "paragraph",
	}
}

// NewChunker 根据策略创建分块器
func NewChunker(config ChunkConfig) ChunkStrategy {
	// 确保配置合理
	if config.MaxSize <= 0 {
		config.MaxSize = 1000
	}
	if config.Overlap < 0 {
		config.Overlap = 100
	}
	if config.Overlap >= config.MaxSize {
		config.Overlap = config.MaxSize / 10 // 默认10%重叠
	}
	if config.MinSize <= 0 {
		config.MinSize = 200
	}

	switch config.Strategy {
	case "sentence":
		return &SentenceChunker{
			config: config,
		}
	case "fixed":
		return &FixedChunker{
			config: config,
		}
	case "paragraph":
		fallthrough
	default:
		return &ParagraphChunker{
			config: config,
		}
	}
}

// ==================== 段落分块器 ====================

// ParagraphChunker 按段落分块（优先策略）
type ParagraphChunker struct {
	config ChunkConfig
}

func (c *ParagraphChunker) Chunk(text string, maxSize, overlap int) []Chunk {
	if maxSize <= 0 {
		maxSize = c.config.MaxSize
	}
	if overlap < 0 {
		overlap = c.config.Overlap
	}

	// 如果文本很短，不需要分块
	if len(text) <= maxSize {
		return []Chunk{
			{
				Text:        text,
				StartIdx:    0,
				EndIdx:      len(text),
				ChunkIdx:    0,
				TotalChunks: 1,
			},
		}
	}

	// 按双换行符分割段落
	paragraphs := splitParagraphs(text)
	if len(paragraphs) == 0 {
		// 如果没有段落分隔符，降级到句子分块
		sentenceChunker := &SentenceChunker{config: c.config}
		return sentenceChunker.Chunk(text, maxSize, overlap)
	}

	chunks := make([]Chunk, 0)
	currentChunk := strings.Builder{}
	currentStart := 0
	chunkIdx := 0

	for i, para := range paragraphs {
		paraText := strings.TrimSpace(para.Text)
		if paraText == "" {
			continue
		}

		// 检查添加这个段落是否会超过限制
		currentLen := currentChunk.Len()
		paraLen := len(paraText)
		
		// 如果当前块为空，直接添加
		if currentLen == 0 {
			currentChunk.WriteString(paraText)
			currentStart = para.StartIdx
			continue
		}

		// 如果添加这个段落会超过限制，且当前块已达到最小大小，则创建新块
		if currentLen+paraLen+1 > maxSize && currentLen >= c.config.MinSize {
			chunks = append(chunks, Chunk{
				Text:        currentChunk.String(),
				StartIdx:    currentStart,
				EndIdx:      paragraphs[i-1].EndIdx,
				ChunkIdx:    chunkIdx,
				TotalChunks: 0, // 稍后更新
			})
			chunkIdx++

			// 处理重叠：从上一个块的末尾开始
			currentChunk.Reset()
			if overlap > 0 && len(chunks) > 0 {
				prevChunk := chunks[len(chunks)-1]
				overlapText := getOverlapText(text, prevChunk.EndIdx-overlap, prevChunk.EndIdx)
				if overlapText != "" {
					currentChunk.WriteString(overlapText)
					currentStart = prevChunk.EndIdx - overlap
				} else {
					currentStart = para.StartIdx
				}
			} else {
				currentStart = para.StartIdx
			}

			currentChunk.WriteString(paraText)
		} else {
			// 添加段落分隔符并继续累积
			if currentChunk.Len() > 0 {
				currentChunk.WriteString("\n\n")
			}
			currentChunk.WriteString(paraText)
		}
	}

	// 添加最后一个块
	if currentChunk.Len() > 0 {
		chunks = append(chunks, Chunk{
			Text:        currentChunk.String(),
			StartIdx:    currentStart,
			EndIdx:      paragraphs[len(paragraphs)-1].EndIdx,
			ChunkIdx:    chunkIdx,
			TotalChunks: 0, // 稍后更新
		})
	}

	// 更新总块数
	totalChunks := len(chunks)
	for i := range chunks {
		chunks[i].TotalChunks = totalChunks
	}

	// 如果只有一个块且超过限制，降级到句子分块
	if len(chunks) == 1 && len(chunks[0].Text) > maxSize {
		sentenceChunker := &SentenceChunker{config: c.config}
		return sentenceChunker.Chunk(text, maxSize, overlap)
	}

	return chunks
}

// paragraphInfo 段落信息
type paragraphInfo struct {
	Text    string
	StartIdx int
	EndIdx   int
}

// splitParagraphs 按段落分割文本
func splitParagraphs(text string) []paragraphInfo {
	// 按双换行符分割
	parts := regexp.MustCompile(`\n\s*\n`).Split(text, -1)
	paragraphs := make([]paragraphInfo, 0)
	currentIdx := 0

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			// 计算跳过的字符数（包括分隔符）
			separatorLen := 2 // \n\n
			if currentIdx+len(part)+separatorLen <= len(text) {
				currentIdx += len(part) + separatorLen
			} else {
				currentIdx = len(text)
			}
			continue
		}

		// 在原文本中查找当前段落的位置
		startIdx := currentIdx
		// 跳过前导空白
		for startIdx < len(text) && (text[startIdx] == '\n' || text[startIdx] == ' ' || text[startIdx] == '\t' || text[startIdx] == '\r') {
			startIdx++
		}

		endIdx := startIdx + len(trimmed)
		if endIdx > len(text) {
			endIdx = len(text)
		}

		paragraphs = append(paragraphs, paragraphInfo{
			Text:     trimmed,
			StartIdx: startIdx,
			EndIdx:   endIdx,
		})

		currentIdx = endIdx
		// 跳过分隔符
		if currentIdx < len(text) && text[currentIdx] == '\n' {
			currentIdx++
			if currentIdx < len(text) && text[currentIdx] == '\n' {
				currentIdx++
			}
		}
	}

	return paragraphs
}

// ==================== 句子分块器 ====================

// SentenceChunker 按句子分块
type SentenceChunker struct {
	config ChunkConfig
}

func (c *SentenceChunker) Chunk(text string, maxSize, overlap int) []Chunk {
	if maxSize <= 0 {
		maxSize = c.config.MaxSize
	}
	if overlap < 0 {
		overlap = c.config.Overlap
	}

	// 如果文本很短，不需要分块
	if len(text) <= maxSize {
		return []Chunk{
			{
				Text:        text,
				StartIdx:    0,
				EndIdx:      len(text),
				ChunkIdx:    0,
				TotalChunks: 1,
			},
		}
	}

	// 按句子分割
	sentences := splitSentences(text)
	if len(sentences) == 0 {
		// 如果没有句子分隔符，降级到固定长度分块
		fixedChunker := &FixedChunker{config: c.config}
		return fixedChunker.Chunk(text, maxSize, overlap)
	}

	chunks := make([]Chunk, 0)
	currentChunk := strings.Builder{}
	currentStart := 0
	chunkIdx := 0

	for i, sent := range sentences {
		sentText := strings.TrimSpace(sent.Text)
		if sentText == "" {
			continue
		}

		currentLen := currentChunk.Len()
		sentLen := len(sentText)

		if currentLen == 0 {
			currentChunk.WriteString(sentText)
			currentStart = sent.StartIdx
			continue
		}

		// 如果添加这个句子会超过限制，且当前块已达到最小大小，则创建新块
		if currentLen+sentLen+1 > maxSize && currentLen >= c.config.MinSize {
			chunks = append(chunks, Chunk{
				Text:        currentChunk.String(),
				StartIdx:    currentStart,
				EndIdx:      sentences[i-1].EndIdx,
				ChunkIdx:    chunkIdx,
				TotalChunks: 0,
			})
			chunkIdx++

			// 处理重叠
			currentChunk.Reset()
			if overlap > 0 && len(chunks) > 0 {
				prevChunk := chunks[len(chunks)-1]
				overlapText := getOverlapText(text, prevChunk.EndIdx-overlap, prevChunk.EndIdx)
				if overlapText != "" {
					currentChunk.WriteString(overlapText)
					currentStart = prevChunk.EndIdx - overlap
				} else {
					currentStart = sent.StartIdx
				}
			} else {
				currentStart = sent.StartIdx
			}

			currentChunk.WriteString(sentText)
		} else {
			if currentChunk.Len() > 0 {
				currentChunk.WriteString("。")
			}
			currentChunk.WriteString(sentText)
		}
	}

	// 添加最后一个块
	if currentChunk.Len() > 0 {
		chunks = append(chunks, Chunk{
			Text:        currentChunk.String(),
			StartIdx:    currentStart,
			EndIdx:      sentences[len(sentences)-1].EndIdx,
			ChunkIdx:    chunkIdx,
			TotalChunks: 0,
		})
	}

	// 更新总块数
	totalChunks := len(chunks)
	for i := range chunks {
		chunks[i].TotalChunks = totalChunks
	}

	// 如果只有一个块且超过限制，降级到固定长度分块
	if len(chunks) == 1 && len(chunks[0].Text) > maxSize {
		fixedChunker := &FixedChunker{config: c.config}
		return fixedChunker.Chunk(text, maxSize, overlap)
	}

	return chunks
}

// sentenceInfo 句子信息
type sentenceInfo struct {
	Text     string
	StartIdx int
	EndIdx   int
}

// splitSentences 按句子分割文本（支持中英文）
var sentenceEndRegex = regexp.MustCompile(`[。！？.!?]\s*`)

func splitSentences(text string) []sentenceInfo {
	matches := sentenceEndRegex.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return []sentenceInfo{{Text: text, StartIdx: 0, EndIdx: len(text)}}
	}

	sentences := make([]sentenceInfo, 0)
	startIdx := 0

	for _, match := range matches {
		endIdx := match[1]
		sentText := strings.TrimSpace(text[startIdx:endIdx])
		if sentText != "" {
			sentences = append(sentences, sentenceInfo{
				Text:     sentText,
				StartIdx: startIdx,
				EndIdx:   endIdx,
			})
		}
		startIdx = endIdx
	}

	// 添加最后一段（如果没有句子结束符）
	if startIdx < len(text) {
		remaining := strings.TrimSpace(text[startIdx:])
		if remaining != "" {
			sentences = append(sentences, sentenceInfo{
				Text:     remaining,
				StartIdx: startIdx,
				EndIdx:   len(text),
			})
		}
	}

	return sentences
}

// ==================== 固定长度分块器 ====================

// FixedChunker 按固定长度分块（最后备选）
type FixedChunker struct {
	config ChunkConfig
}

func (c *FixedChunker) Chunk(text string, maxSize, overlap int) []Chunk {
	if maxSize <= 0 {
		maxSize = c.config.MaxSize
	}
	if overlap < 0 {
		overlap = c.config.Overlap
	}

	// 如果文本很短，不需要分块
	if len(text) <= maxSize {
		return []Chunk{
			{
				Text:        text,
				StartIdx:    0,
				EndIdx:      len(text),
				ChunkIdx:    0,
				TotalChunks: 1,
			},
		}
	}

	chunks := make([]Chunk, 0)
	startIdx := 0
	chunkIdx := 0

	for startIdx < len(text) {
		endIdx := startIdx + maxSize
		if endIdx > len(text) {
			endIdx = len(text)
		}

		chunkText := text[startIdx:endIdx]

		// 尝试在单词边界处截断（避免截断单词）
		if endIdx < len(text) {
			chunkText = truncateAtBoundary(chunkText, maxSize)
			endIdx = startIdx + len(chunkText)
		}

		chunks = append(chunks, Chunk{
			Text:        chunkText,
			StartIdx:    startIdx,
			EndIdx:      endIdx,
			ChunkIdx:    chunkIdx,
			TotalChunks: 0, // 稍后更新
		})

		chunkIdx++
		startIdx = endIdx - overlap // 应用重叠
		if startIdx < 0 {
			startIdx = 0
		}
	}

	// 更新总块数
	totalChunks := len(chunks)
	for i := range chunks {
		chunks[i].TotalChunks = totalChunks
	}

	return chunks
}

// truncateAtBoundary 在边界处截断（避免截断单词或中文字符）
func truncateAtBoundary(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	// 尝试在空格、标点符号处截断
	boundaryChars := []rune{' ', '\n', '\t', '。', '，', '！', '？', '.', ',', '!', '?'}
	textRunes := []rune(text)
	
	if len(textRunes) <= maxLen {
		return text
	}

	// 从后往前找边界字符
	for i := maxLen - 1; i >= maxLen/2; i-- {
		for _, boundary := range boundaryChars {
			if textRunes[i] == boundary {
				return string(textRunes[:i+1])
			}
		}
	}

	// 如果找不到边界，直接截断
	return string(textRunes[:maxLen])
}

// getOverlapText 获取重叠文本
func getOverlapText(text string, startIdx, endIdx int) string {
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx > len(text) {
		endIdx = len(text)
	}
	if startIdx >= endIdx {
		return ""
	}
	return text[startIdx:endIdx]
}

