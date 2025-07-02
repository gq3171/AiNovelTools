package novel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// NovelManager 小说写作管理器
type NovelManager struct {
	projectPath    string
	novelData      *NovelProject
	chatHistory    []ChatRecord
	contentIndex   *ContentIndex
	mutex          sync.RWMutex
}

// NovelProject 小说项目数据
type NovelProject struct {
	Title         string            `json:"title"`
	Author        string            `json:"author"`
	Genre         string            `json:"genre"`
	CreatedAt     time.Time         `json:"created_at"`
	LastModified  time.Time         `json:"last_modified"`
	
	// 核心设定
	Characters    map[string]*Character    `json:"characters"`
	WorldSettings map[string]*WorldSetting `json:"world_settings"`
	PlotLines     map[string]*PlotLine     `json:"plot_lines"`
	
	// 章节管理
	Chapters      []*Chapter        `json:"chapters"`
	CurrentChapter int              `json:"current_chapter"`
	
	// 写作设置
	WritingStyle  WritingStyle      `json:"writing_style"`
	TargetWords   int               `json:"target_words"`
	
	// 元数据
	Tags          []string          `json:"tags"`
	Notes         []string          `json:"notes"`
}

// Character 角色设定
type Character struct {
	Name          string            `json:"name"`
	Age           int               `json:"age"`
	Gender        string            `json:"gender"`
	Occupation    string            `json:"occupation"`
	Personality   []string          `json:"personality"`
	Appearance    string            `json:"appearance"`
	Background    string            `json:"background"`
	Relationships map[string]string `json:"relationships"`
	
	// 角色发展
	CharacterArc  []string          `json:"character_arc"`
	KeyDialogues  []string          `json:"key_dialogues"`
	FirstAppeared int               `json:"first_appeared"` // 章节号
	LastAppeared  int               `json:"last_appeared"`
}

// WorldSetting 世界观设定
type WorldSetting struct {
	Category     string    `json:"category"` // 地理、魔法、科技、社会等
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Rules        []string  `json:"rules"`
	RelatedItems []string  `json:"related_items"`
	FirstMentioned int     `json:"first_mentioned"` // 章节号
}

// PlotLine 情节线
type PlotLine struct {
	Name         string     `json:"name"`
	Type         string     `json:"type"` // main, sub, romance, mystery等
	Status       string     `json:"status"` // active, resolved, suspended
	StartChapter int        `json:"start_chapter"`
	EndChapter   int        `json:"end_chapter"`
	KeyEvents    []PlotEvent `json:"key_events"`
	Foreshadowing []string   `json:"foreshadowing"` // 伏笔
}

// PlotEvent 情节事件
type PlotEvent struct {
	Chapter     int       `json:"chapter"`
	Description string    `json:"description"`
	Characters  []string  `json:"characters"`
	Significance string   `json:"significance"`
	Timestamp   time.Time `json:"timestamp"`
}

// Chapter 章节信息
type Chapter struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	WordCount   int       `json:"word_count"`
	Status      string    `json:"status"` // draft, reviewing, completed
	WrittenAt   time.Time `json:"written_at"`
	
	// 内容分析
	Characters  []string  `json:"characters"` // 本章出现的角色
	PlotLines   []string  `json:"plot_lines"` // 本章涉及的情节线
	KeyEvents   []string  `json:"key_events"` // 本章关键事件
	Emotions    []string  `json:"emotions"`   // 情感基调
}

// WritingStyle 写作风格
type WritingStyle struct {
	Perspective string   `json:"perspective"` // first, third_limited, third_omniscient
	Tense       string   `json:"tense"`       // past, present
	Voice       string   `json:"voice"`       // formal, casual, poetic
	Themes      []string `json:"themes"`
	ToneKeywords []string `json:"tone_keywords"`
}

// ChatRecord 聊天记录
type ChatRecord struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	ChapterNum  int       `json:"chapter_num"`
	UserMessage string    `json:"user_message"`
	AIResponse  string    `json:"ai_response"`
	
	// 内容分析
	Intent      string    `json:"intent"`      // writing, editing, plotting, character_dev
	Mentions    Mentions  `json:"mentions"`    // 提到的角色、设定等
	Decisions   []string  `json:"decisions"`   // 做出的创作决定
}

// Mentions 提及的内容
type Mentions struct {
	Characters    []string `json:"characters"`
	WorldSettings []string `json:"world_settings"`
	PlotLines     []string `json:"plot_lines"`
	Chapters      []int    `json:"chapters"`
}

// ContentIndex 内容索引
type ContentIndex struct {
	CharacterIndex    map[string][]int `json:"character_index"`    // 角色名 -> 出现的聊天记录ID
	SettingIndex      map[string][]int `json:"setting_index"`      // 设定 -> 相关聊天记录
	PlotIndex         map[string][]int `json:"plot_index"`         // 情节 -> 相关记录
	KeywordIndex      map[string][]int `json:"keyword_index"`      // 关键词 -> 记录
}

// NewNovelManager 创建小说管理器
func NewNovelManager(projectPath string) *NovelManager {
	return &NovelManager{
		projectPath:  projectPath,
		chatHistory:  make([]ChatRecord, 0),
		contentIndex: &ContentIndex{
			CharacterIndex: make(map[string][]int),
			SettingIndex:   make(map[string][]int),
			PlotIndex:      make(map[string][]int),
			KeywordIndex:   make(map[string][]int),
		},
	}
}

// InitializeProject 初始化小说项目
func (nm *NovelManager) InitializeProject(title, author, genre string) error {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()
	
	nm.novelData = &NovelProject{
		Title:         title,
		Author:        author,
		Genre:         genre,
		CreatedAt:     time.Now(),
		LastModified:  time.Now(),
		Characters:    make(map[string]*Character),
		WorldSettings: make(map[string]*WorldSetting),
		PlotLines:     make(map[string]*PlotLine),
		Chapters:      make([]*Chapter, 0),
		CurrentChapter: 0,
		WritingStyle: WritingStyle{
			Perspective: "third_limited",
			Tense:       "past",
			Voice:       "casual",
			Themes:      make([]string, 0),
			ToneKeywords: make([]string, 0),
		},
		TargetWords: 100000,
		Tags:        make([]string, 0),
		Notes:       make([]string, 0),
	}
	
	return nm.SaveProject()
}

// LoadProject 加载项目
func (nm *NovelManager) LoadProject() error {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()
	
	projectFile := filepath.Join(nm.projectPath, "novel_project.json")
	historyFile := filepath.Join(nm.projectPath, "chat_history.json")
	indexFile := filepath.Join(nm.projectPath, "content_index.json")
	
	// 加载项目数据
	if data, err := os.ReadFile(projectFile); err == nil {
		if err := json.Unmarshal(data, &nm.novelData); err != nil {
			return fmt.Errorf("failed to parse project file: %w", err)
		}
	}
	
	// 加载聊天历史
	if data, err := os.ReadFile(historyFile); err == nil {
		if err := json.Unmarshal(data, &nm.chatHistory); err != nil {
			return fmt.Errorf("failed to parse chat history: %w", err)
		}
	}
	
	// 加载内容索引
	if data, err := os.ReadFile(indexFile); err == nil {
		if err := json.Unmarshal(data, &nm.contentIndex); err != nil {
			return fmt.Errorf("failed to parse content index: %w", err)
		}
	}
	
	return nil
}

// SaveProject 保存项目
func (nm *NovelManager) SaveProject() error {
	// 确保目录存在
	if err := os.MkdirAll(nm.projectPath, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}
	
	// 保存项目数据
	if nm.novelData != nil {
		nm.novelData.LastModified = time.Now()
		projectFile := filepath.Join(nm.projectPath, "novel_project.json")
		data, _ := json.MarshalIndent(nm.novelData, "", "  ")
		if err := os.WriteFile(projectFile, data, 0644); err != nil {
			return fmt.Errorf("failed to save project file: %w", err)
		}
	}
	
	// 保存聊天历史
	historyFile := filepath.Join(nm.projectPath, "chat_history.json")
	historyData, _ := json.MarshalIndent(nm.chatHistory, "", "  ")
	if err := os.WriteFile(historyFile, historyData, 0644); err != nil {
		return fmt.Errorf("failed to save chat history: %w", err)
	}
	
	// 保存内容索引
	indexFile := filepath.Join(nm.projectPath, "content_index.json")
	indexData, _ := json.MarshalIndent(nm.contentIndex, "", "  ")
	if err := os.WriteFile(indexFile, indexData, 0644); err != nil {
		return fmt.Errorf("failed to save content index: %w", err)
	}
	
	return nil
}

// AddChatRecord 添加聊天记录
func (nm *NovelManager) AddChatRecord(userMsg, aiResponse string, chapterNum int) error {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()
	
	record := ChatRecord{
		ID:          fmt.Sprintf("chat_%d", time.Now().Unix()),
		Timestamp:   time.Now(),
		ChapterNum:  chapterNum,
		UserMessage: userMsg,
		AIResponse:  aiResponse,
		Intent:      nm.analyzeIntent(userMsg),
		Mentions:    nm.extractMentions(userMsg + " " + aiResponse),
		Decisions:   nm.extractDecisions(aiResponse),
	}
	
	nm.chatHistory = append(nm.chatHistory, record)
	nm.updateContentIndex(record)
	
	// 自动保存
	go nm.SaveProject()
	
	return nil
}

// GetRelevantHistory 获取相关历史记录
func (nm *NovelManager) GetRelevantHistory(query string, maxRecords int) []ChatRecord {
	nm.mutex.RLock()
	defer nm.mutex.RUnlock()
	
	relevantRecords := make([]ChatRecord, 0)
	queryLower := strings.ToLower(query)
	
	// 搜索算法：关键词匹配 + 时间权重
	for i := len(nm.chatHistory) - 1; i >= 0 && len(relevantRecords) < maxRecords; i-- {
		record := nm.chatHistory[i]
		
		content := strings.ToLower(record.UserMessage + " " + record.AIResponse)
		if strings.Contains(content, queryLower) {
			relevantRecords = append(relevantRecords, record)
		}
	}
	
	return relevantRecords
}

// GetChapterContext 获取章节上下文
func (nm *NovelManager) GetChapterContext(chapterNum int) string {
	nm.mutex.RLock()
	defer nm.mutex.RUnlock()
	
	var context strings.Builder
	context.WriteString(fmt.Sprintf("=== 第%d章 写作上下文 ===\n\n", chapterNum))
	
	// 章节信息
	if chapterNum > 0 && chapterNum <= len(nm.novelData.Chapters) {
		chapter := nm.novelData.Chapters[chapterNum-1]
		context.WriteString(fmt.Sprintf("章节标题: %s\n", chapter.Title))
		context.WriteString(fmt.Sprintf("章节概要: %s\n", chapter.Summary))
		context.WriteString(fmt.Sprintf("涉及角色: %s\n", strings.Join(chapter.Characters, ", ")))
		context.WriteString(fmt.Sprintf("相关情节: %s\n\n", strings.Join(chapter.PlotLines, ", ")))
	}
	
	// 相关角色信息
	context.WriteString("=== 相关角色 ===\n")
	for name, char := range nm.novelData.Characters {
		if char.FirstAppeared <= chapterNum && (char.LastAppeared == 0 || char.LastAppeared >= chapterNum) {
			context.WriteString(fmt.Sprintf("• %s: %s\n", name, char.Background))
		}
	}
	
	// 活跃情节线
	context.WriteString("\n=== 活跃情节线 ===\n")
	for name, plot := range nm.novelData.PlotLines {
		if plot.StartChapter <= chapterNum && (plot.EndChapter == 0 || plot.EndChapter >= chapterNum) {
			context.WriteString(fmt.Sprintf("• %s (%s): %s\n", name, plot.Status, getLastEvent(plot)))
		}
	}
	
	// 最近的聊天记录
	context.WriteString("\n=== 最近讨论 ===\n")
	recentChats := nm.getChapterChats(chapterNum, 5)
	for _, chat := range recentChats {
		context.WriteString(fmt.Sprintf("• %s: %s\n", 
			chat.Timestamp.Format("01-02 15:04"), 
			truncateString(chat.UserMessage, 100)))
	}
	
	return context.String()
}

// 辅助函数
func (nm *NovelManager) analyzeIntent(message string) string {
	msgLower := strings.ToLower(message)
	
	if strings.Contains(msgLower, "写") || strings.Contains(msgLower, "继续") {
		return "writing"
	}
	if strings.Contains(msgLower, "修改") || strings.Contains(msgLower, "改") {
		return "editing"
	}
	if strings.Contains(msgLower, "角色") || strings.Contains(msgLower, "人物") {
		return "character_dev"
	}
	if strings.Contains(msgLower, "情节") || strings.Contains(msgLower, "剧情") {
		return "plotting"
	}
	
	return "general"
}

func (nm *NovelManager) extractMentions(text string) Mentions {
	mentions := Mentions{
		Characters:    make([]string, 0),
		WorldSettings: make([]string, 0),
		PlotLines:     make([]string, 0),
		Chapters:      make([]int, 0),
	}
	
	if nm.novelData == nil {
		return mentions
	}
	
	textLower := strings.ToLower(text)
	
	// 提取角色提及
	for charName := range nm.novelData.Characters {
		if strings.Contains(textLower, strings.ToLower(charName)) {
			mentions.Characters = append(mentions.Characters, charName)
		}
	}
	
	// 提取设定提及
	for settingName := range nm.novelData.WorldSettings {
		if strings.Contains(textLower, strings.ToLower(settingName)) {
			mentions.WorldSettings = append(mentions.WorldSettings, settingName)
		}
	}
	
	// 提取情节线提及
	for plotName := range nm.novelData.PlotLines {
		if strings.Contains(textLower, strings.ToLower(plotName)) {
			mentions.PlotLines = append(mentions.PlotLines, plotName)
		}
	}
	
	return mentions
}

func (nm *NovelManager) extractDecisions(aiResponse string) []string {
	decisions := make([]string, 0)
	
	// 简单的决定提取逻辑
	lines := strings.Split(aiResponse, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "决定") || strings.Contains(line, "设定") || 
		   strings.Contains(line, "让") || strings.Contains(line, "应该") {
			if len(line) > 10 && len(line) < 100 {
				decisions = append(decisions, line)
			}
		}
	}
	
	return decisions
}

func (nm *NovelManager) updateContentIndex(record ChatRecord) {
	// 更新角色索引
	for _, char := range record.Mentions.Characters {
		nm.contentIndex.CharacterIndex[char] = append(
			nm.contentIndex.CharacterIndex[char], 
			len(nm.chatHistory)-1)
	}
	
	// 更新设定索引
	for _, setting := range record.Mentions.WorldSettings {
		nm.contentIndex.SettingIndex[setting] = append(
			nm.contentIndex.SettingIndex[setting], 
			len(nm.chatHistory)-1)
	}
	
	// 更新情节索引
	for _, plot := range record.Mentions.PlotLines {
		nm.contentIndex.PlotIndex[plot] = append(
			nm.contentIndex.PlotIndex[plot], 
			len(nm.chatHistory)-1)
	}
}

func (nm *NovelManager) getChapterChats(chapterNum, limit int) []ChatRecord {
	chats := make([]ChatRecord, 0)
	count := 0
	
	for i := len(nm.chatHistory) - 1; i >= 0 && count < limit; i-- {
		if nm.chatHistory[i].ChapterNum == chapterNum {
			chats = append(chats, nm.chatHistory[i])
			count++
		}
	}
	
	return chats
}

func getLastEvent(plot *PlotLine) string {
	if len(plot.KeyEvents) > 0 {
		return plot.KeyEvents[len(plot.KeyEvents)-1].Description
	}
	return "暂无事件"
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}