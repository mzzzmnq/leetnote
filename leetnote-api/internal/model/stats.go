package model

import "time"

// StatsOverview 是刷题概览统计。
type StatsOverview struct {
	TotalNotes     int64
	TotalProblems  int64 // 去重后的题目数
	TotalSolutions int64
	TotalTags      int64

	// 按难度分布（只统计关联了题目的笔记）
	Easy   int64
	Medium int64
	Hard   int64

	Starred int64
	Drafts  int64

	ActiveDays    int64 // 有记录的天数
	CurrentStreak int   // 当前连续打卡天数
	LongestStreak int   // 历史最长连续打卡
}

// TrendPoint 是趋势图上的一个点：某一天新增了多少篇笔记。
type TrendPoint struct {
	Day   time.Time
	Count int64
}
