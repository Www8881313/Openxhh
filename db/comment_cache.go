package db

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"openxhh/loger"
	"openxhh/pg"
	"openxhh/sqlite"

	"go.uber.org/zap"
)

type CommentCachePost struct {
	LinkID int64
	Title  string
}

type CommentCacheItem struct {
	LinkID        int64
	RootCommentID int64
	CommentID     int64
	ReplyID       int64
	FloorNum      int64
	UserID        int64
	UserName      string
	AvatarURL     string
	ReplyUserName string
	CreatedAt     string
	Text          string
	Images        []string
}

func migrateCommentCacheTables() {
	ctx := context.Background()
	if cfg.Type == "pg" {
		_, err := pg.Conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS xhh_post_cache (
			link_id BIGINT PRIMARY KEY,
			title TEXT DEFAULT '',
			updated_at BIGINT DEFAULT 0
		)`)
		if err != nil {
			loger.Loger.Warn("[DB]无法创建帖子缓存表", zap.Error(err))
		}
		_, err = pg.Conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS xhh_comment_cache (
			comment_id BIGINT PRIMARY KEY,
			link_id BIGINT DEFAULT 0,
			root_comment_id BIGINT DEFAULT 0,
			reply_id BIGINT DEFAULT 0,
			floor_num BIGINT DEFAULT 0,
			user_id BIGINT DEFAULT 0,
			user_name TEXT DEFAULT '',
			avatar_url TEXT DEFAULT '',
			reply_user_name TEXT DEFAULT '',
			created_at TEXT DEFAULT '',
			text TEXT DEFAULT '',
			images TEXT DEFAULT '[]',
			updated_at BIGINT DEFAULT 0
		)`)
		if err != nil {
			loger.Loger.Warn("[DB]无法创建评论缓存表", zap.Error(err))
		}
		_, err = pg.Conn.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_xhh_comment_cache_floor ON xhh_comment_cache (link_id, root_comment_id)`)
		if err != nil {
			loger.Loger.Warn("[DB]无法创建评论缓存索引", zap.Error(err))
		}
	}
	if cfg.Type == "sqlite" {
		_, err := sqlite.Db.Exec(`CREATE TABLE IF NOT EXISTS xhh_post_cache (
			link_id BIGINT PRIMARY KEY,
			title TEXT DEFAULT '',
			updated_at BIGINT DEFAULT 0
		)`)
		if err != nil {
			loger.Loger.Warn("[DB]无法创建帖子缓存表", zap.Error(err))
		}
		_, err = sqlite.Db.Exec(`CREATE TABLE IF NOT EXISTS xhh_comment_cache (
			comment_id BIGINT PRIMARY KEY,
			link_id BIGINT DEFAULT 0,
			root_comment_id BIGINT DEFAULT 0,
			reply_id BIGINT DEFAULT 0,
			floor_num BIGINT DEFAULT 0,
			user_id BIGINT DEFAULT 0,
			user_name TEXT DEFAULT '',
			avatar_url TEXT DEFAULT '',
			reply_user_name TEXT DEFAULT '',
			created_at TEXT DEFAULT '',
			text TEXT DEFAULT '',
			images TEXT DEFAULT '[]',
			updated_at BIGINT DEFAULT 0
		)`)
		if err != nil {
			loger.Loger.Warn("[DB]无法创建评论缓存表", zap.Error(err))
		}
		_, err = sqlite.Db.Exec(`CREATE INDEX IF NOT EXISTS idx_xhh_comment_cache_floor ON xhh_comment_cache (link_id, root_comment_id)`)
		if err != nil {
			loger.Loger.Warn("[DB]无法创建评论缓存索引", zap.Error(err))
		}
	}
}

func SaveCommentThreadCache(post CommentCachePost, comments []CommentCacheItem) bool {
	if !commentCacheDatabaseReady() {
		return false
	}
	updatedAt := time.Now().Unix()
	ok := true
	if post.LinkID > 0 {
		post.Title = strings.TrimSpace(post.Title)
		if !saveCommentCachePost(post, updatedAt) {
			ok = false
		}
	}
	for _, comment := range comments {
		if comment.CommentID <= 0 {
			continue
		}
		if comment.RootCommentID <= 0 {
			comment.RootCommentID = comment.CommentID
		}
		if !saveCommentCacheItem(comment, updatedAt) {
			ok = false
		}
	}
	return ok
}

func saveCommentCachePost(post CommentCachePost, updatedAt int64) bool {
	ctx := context.Background()
	if cfg.Type == "pg" {
		_, err := pg.Conn.Exec(ctx, `INSERT INTO xhh_post_cache (link_id,title,updated_at)
			VALUES ($1,$2,$3)
			ON CONFLICT (link_id) DO UPDATE SET title=EXCLUDED.title, updated_at=EXCLUDED.updated_at`, post.LinkID, post.Title, updatedAt)
		if err != nil {
			loger.Loger.Warn("[DB]无法保存帖子缓存", zap.Error(err), zap.Int64("link_id", post.LinkID))
			return false
		}
		return true
	}
	if cfg.Type == "sqlite" {
		_, err := sqlite.Db.Exec(`INSERT INTO xhh_post_cache (link_id,title,updated_at)
			VALUES (?,?,?)
			ON CONFLICT (link_id) DO UPDATE SET title=excluded.title, updated_at=excluded.updated_at`, post.LinkID, post.Title, updatedAt)
		if err != nil {
			loger.Loger.Warn("[DB]无法保存帖子缓存", zap.Error(err), zap.Int64("link_id", post.LinkID))
			return false
		}
		return true
	}
	return false
}

func saveCommentCacheItem(comment CommentCacheItem, updatedAt int64) bool {
	images, err := json.Marshal(comment.Images)
	if err != nil {
		images = []byte("[]")
	}
	comment.UserName = strings.TrimSpace(comment.UserName)
	comment.AvatarURL = strings.TrimSpace(comment.AvatarURL)
	comment.ReplyUserName = strings.TrimSpace(comment.ReplyUserName)
	comment.CreatedAt = strings.TrimSpace(comment.CreatedAt)
	comment.Text = strings.TrimSpace(comment.Text)
	ctx := context.Background()
	if cfg.Type == "pg" {
		_, err := pg.Conn.Exec(ctx, `INSERT INTO xhh_comment_cache (comment_id,link_id,root_comment_id,reply_id,floor_num,user_id,user_name,avatar_url,reply_user_name,created_at,text,images,updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
			ON CONFLICT (comment_id) DO UPDATE SET
			link_id=CASE WHEN EXCLUDED.link_id > 0 THEN EXCLUDED.link_id ELSE xhh_comment_cache.link_id END,
			root_comment_id=CASE WHEN EXCLUDED.root_comment_id > 0 THEN EXCLUDED.root_comment_id ELSE xhh_comment_cache.root_comment_id END,
			reply_id=EXCLUDED.reply_id,
			floor_num=CASE WHEN EXCLUDED.floor_num > 0 THEN EXCLUDED.floor_num ELSE xhh_comment_cache.floor_num END,
			user_id=CASE WHEN EXCLUDED.user_id > 0 THEN EXCLUDED.user_id ELSE xhh_comment_cache.user_id END,
			user_name=CASE WHEN EXCLUDED.user_name <> '' THEN EXCLUDED.user_name ELSE xhh_comment_cache.user_name END,
			avatar_url=CASE WHEN EXCLUDED.avatar_url <> '' THEN EXCLUDED.avatar_url ELSE xhh_comment_cache.avatar_url END,
			reply_user_name=CASE WHEN EXCLUDED.reply_user_name <> '' THEN EXCLUDED.reply_user_name ELSE xhh_comment_cache.reply_user_name END,
			created_at=CASE WHEN EXCLUDED.created_at <> '' THEN EXCLUDED.created_at ELSE xhh_comment_cache.created_at END,
			text=CASE WHEN EXCLUDED.text <> '' THEN EXCLUDED.text ELSE xhh_comment_cache.text END,
			images=CASE WHEN EXCLUDED.images <> '[]' THEN EXCLUDED.images ELSE xhh_comment_cache.images END,
			updated_at=EXCLUDED.updated_at`, comment.CommentID, comment.LinkID, comment.RootCommentID, comment.ReplyID, comment.FloorNum, comment.UserID, comment.UserName, comment.AvatarURL, comment.ReplyUserName, comment.CreatedAt, comment.Text, string(images), updatedAt)
		if err != nil {
			loger.Loger.Warn("[DB]无法保存评论缓存", zap.Error(err), zap.Int64("comment_id", comment.CommentID))
			return false
		}
		return true
	}
	if cfg.Type == "sqlite" {
		_, err := sqlite.Db.Exec(`INSERT INTO xhh_comment_cache (comment_id,link_id,root_comment_id,reply_id,floor_num,user_id,user_name,avatar_url,reply_user_name,created_at,text,images,updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
			ON CONFLICT (comment_id) DO UPDATE SET
			link_id=CASE WHEN excluded.link_id > 0 THEN excluded.link_id ELSE xhh_comment_cache.link_id END,
			root_comment_id=CASE WHEN excluded.root_comment_id > 0 THEN excluded.root_comment_id ELSE xhh_comment_cache.root_comment_id END,
			reply_id=excluded.reply_id,
			floor_num=CASE WHEN excluded.floor_num > 0 THEN excluded.floor_num ELSE xhh_comment_cache.floor_num END,
			user_id=CASE WHEN excluded.user_id > 0 THEN excluded.user_id ELSE xhh_comment_cache.user_id END,
			user_name=CASE WHEN excluded.user_name <> '' THEN excluded.user_name ELSE xhh_comment_cache.user_name END,
			avatar_url=CASE WHEN excluded.avatar_url <> '' THEN excluded.avatar_url ELSE xhh_comment_cache.avatar_url END,
			reply_user_name=CASE WHEN excluded.reply_user_name <> '' THEN excluded.reply_user_name ELSE xhh_comment_cache.reply_user_name END,
			created_at=CASE WHEN excluded.created_at <> '' THEN excluded.created_at ELSE xhh_comment_cache.created_at END,
			text=CASE WHEN excluded.text <> '' THEN excluded.text ELSE xhh_comment_cache.text END,
			images=CASE WHEN excluded.images <> '[]' THEN excluded.images ELSE xhh_comment_cache.images END,
			updated_at=excluded.updated_at`, comment.CommentID, comment.LinkID, comment.RootCommentID, comment.ReplyID, comment.FloorNum, comment.UserID, comment.UserName, comment.AvatarURL, comment.ReplyUserName, comment.CreatedAt, comment.Text, string(images), updatedAt)
		if err != nil {
			loger.Loger.Warn("[DB]无法保存评论缓存", zap.Error(err), zap.Int64("comment_id", comment.CommentID))
			return false
		}
		return true
	}
	return false
}

func CachedSubCommentItems(rootCommentID int) []CommentCacheItem {
	if !commentCacheDatabaseReady() || rootCommentID <= 0 {
		return nil
	}
	ctx := context.Background()
	if cfg.Type == "pg" {
		rows, err := pg.Conn.Query(ctx, `SELECT comment_id,root_comment_id,reply_id,floor_num,user_id,COALESCE(user_name,''),COALESCE(text,''),COALESCE(created_at,'')
			FROM xhh_comment_cache WHERE root_comment_id=$1 AND comment_id<>$1 ORDER BY comment_id ASC`, rootCommentID)
		if err != nil {
			loger.Loger.Warn("[DB]无法查询缓存子评论", zap.Error(err), zap.Int("root_comment_id", rootCommentID))
			return nil
		}
		defer rows.Close()
		var items []CommentCacheItem
		for rows.Next() {
			var c CommentCacheItem
			if err := rows.Scan(&c.CommentID, &c.RootCommentID, &c.ReplyID, &c.FloorNum, &c.UserID, &c.UserName, &c.Text, &c.CreatedAt); err != nil {
				continue
			}
			items = append(items, c)
		}
		return items
	}
	if cfg.Type == "sqlite" {
		rows, err := sqlite.Db.Query(`SELECT comment_id,root_comment_id,reply_id,floor_num,user_id,COALESCE(user_name,''),COALESCE(text,''),COALESCE(created_at,'')
			FROM xhh_comment_cache WHERE root_comment_id=? AND comment_id<>? ORDER BY comment_id ASC`, rootCommentID, rootCommentID)
		if err != nil {
			loger.Loger.Warn("[DB]无法查询缓存子评论", zap.Error(err), zap.Int("root_comment_id", rootCommentID))
			return nil
		}
		defer rows.Close()
		var items []CommentCacheItem
		for rows.Next() {
			var c CommentCacheItem
			if err := rows.Scan(&c.CommentID, &c.RootCommentID, &c.ReplyID, &c.FloorNum, &c.UserID, &c.UserName, &c.Text, &c.CreatedAt); err != nil {
				continue
			}
			items = append(items, c)
		}
		return items
	}
	return nil
}

func CleanupCommentCache(maxAgeSeconds int64) int64 {
	if !commentCacheDatabaseReady() || maxAgeSeconds <= 0 {
		return 0
	}
	cutoff := time.Now().Unix() - maxAgeSeconds
	ctx := context.Background()
	if cfg.Type == "pg" {
		tag, err := pg.Conn.Exec(ctx, `DELETE FROM xhh_comment_cache WHERE updated_at > 0 AND updated_at < $1`, cutoff)
		if err != nil {
			loger.Loger.Warn("[DB]无法清理评论缓存", zap.Error(err))
			return 0
		}
		return tag.RowsAffected()
	}
	if cfg.Type == "sqlite" {
		res, err := sqlite.Db.Exec(`DELETE FROM xhh_comment_cache WHERE updated_at > 0 AND updated_at < ?`, cutoff)
		if err != nil {
			loger.Loger.Warn("[DB]无法清理评论缓存", zap.Error(err))
			return 0
		}
		n, _ := res.RowsAffected()
		return n
	}
	return 0
}

func commentCacheDatabaseReady() bool {
	if cfg.Type == "pg" {
		return pg.Conn != nil
	}
	if cfg.Type == "sqlite" {
		return sqlite.Db != nil
	}
	return false
}
