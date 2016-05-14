// Package v1 implements the first version of the Ripple API.
package v1

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"git.zxq.co/ripple/rippleapi/common"
)

type userData struct {
	common.ResponseBase
	ID             int       `json:"id"`
	Username       string    `json:"username"`
	UsernameAKA    string    `json:"username_aka"`
	RegisteredOn   time.Time `json:"registered_on"`
	Rank           int       `json:"rank"`
	LatestActivity time.Time `json:"latest_activity"`
	Country        string    `json:"country"`
}

// UsersGET is the API handler for GET /users
func UsersGET(md common.MethodData) common.CodeMessager {
	var err error
	var whereClause string
	var param interface{}

	switch {
	case md.C.Query("id") == "self":
		param = md.ID()
		whereClause = "users.id = ?"
	case md.C.Query("id") != "":
		param, err = strconv.Atoi(md.C.Query("id"))
		if err != nil {
			return common.SimpleResponse(400, "passed user ID is not a valid number")
		}
		whereClause = "users.id = ?"
	case md.C.Query("name") != "":
		param = md.C.Query("name")
		whereClause = "users.username = ?"
	default:
		return common.SimpleResponse(400, "must provide either querystring param id or param name")
	}

	query := `
SELECT users.id, users.username, register_datetime, rank,
	latest_activity, users_stats.username_aka,
	users_stats.country, users_stats.show_country
FROM users
LEFT JOIN users_stats
ON users.id=users_stats.id
WHERE ` + whereClause + ` AND users.allowed='1'
LIMIT 1`
	return userPuts(md, md.DB.QueryRow(query, param))
}

func userPuts(md common.MethodData, row *sql.Row) common.CodeMessager {
	var err error
	var user userData

	var (
		registeredOn   int64
		latestActivity int64
		showCountry    bool
	)
	err = row.Scan(&user.ID, &user.Username, &registeredOn, &user.Rank, &latestActivity, &user.UsernameAKA, &user.Country, &showCountry)
	switch {
	case err == sql.ErrNoRows:
		return common.SimpleResponse(404, "No such user was found!")
	case err != nil:
		md.Err(err)
		return Err500
	}

	user.RegisteredOn = time.Unix(registeredOn, 0)
	user.LatestActivity = time.Unix(latestActivity, 0)

	user.Country = genCountry(md, user.ID, showCountry, user.Country)

	user.Code = 200
	return user
}

func badgesToArray(badges string) []int {
	var end []int
	badgesSl := strings.Split(badges, ",")
	for _, badge := range badgesSl {
		if badge != "" && badge != "0" {
			// We are ignoring errors because who really gives a shit if something's gone wrong on our end in this
			// particular thing, we can just silently ignore this.
			nb, err := strconv.Atoi(badge)
			if err == nil && nb != 0 {
				end = append(end, nb)
			}
		}
	}
	return end
}

func genCountry(md common.MethodData, uid int, showCountry bool, country string) string {
	// If the user wants to stay anonymous, don't show their country.
	// This can be overriden if we have the ReadConfidential privilege and the user we are accessing is the token owner.
	if showCountry || (md.User.Privileges.HasPrivilegeReadConfidential() && uid == md.ID()) {
		return country
	}
	return "XX"
}

// UserSelfGET is a shortcut for /users/id/self. (/users/self)
func UserSelfGET(md common.MethodData) common.CodeMessager {
	md.C.Request.URL.RawQuery = "id=self&" + md.C.Request.URL.RawQuery
	return UsersGET(md)
}

type whatIDResponse struct {
	common.ResponseBase
	ID int `json:"id"`
}

// UserWhatsTheIDGET is an API request that only returns an user's ID.
func UserWhatsTheIDGET(md common.MethodData) common.CodeMessager {
	var (
		r       whatIDResponse
		allowed int
	)
	err := md.DB.QueryRow("SELECT id, allowed FROM users WHERE username = ? LIMIT 1", md.C.Param("username")).Scan(&r.ID, &allowed)
	if err != nil || (allowed != 1 && !md.User.Privileges.HasPrivilegeViewUserAdvanced()) {
		return common.SimpleResponse(404, "That user could not be found!")
	}
	r.Code = 200
	return r
}

type modeData struct {
	RankedScore           uint64  `json:"ranked_score"`
	TotalScore            uint64  `json:"total_score"`
	PlayCount             int     `json:"playcount"`
	ReplaysWatched        int     `json:"replays_watched"`
	TotalHits             int     `json:"total_hits"`
	Level                 float64 `json:"level"`
	Accuracy              float64 `json:"accuracy"`
	GlobalLeaderboardRank int     `json:"global_leaderboard_rank"`
}
type userFullResponse struct {
	common.ResponseBase
	userData
	STD           modeData `json:"std"`
	Taiko         modeData `json:"taiko"`
	CTB           modeData `json:"ctb"`
	Mania         modeData `json:"mania"`
	PlayStyle     int      `json:"play_style"`
	FavouriteMode int      `json:"favourite_mode"`
	Badges        []int    `json:"badges"`
}

// UserFullGET gets all of an user's information, with one exception: their userpage.
func UserFullGET(md common.MethodData) common.CodeMessager {
	// Hellest query I've ever done.
	query := `
SELECT
	users.id, users.username, users.register_datetime, users.rank, users.latest_activity,
	
	users_stats.username_aka, users_stats.badges_shown, users_stats.country, users_stats.show_country,
	users_stats.play_style, users_stats.favourite_mode,
	
	users_stats.ranked_score_std, users_stats.total_score_std, users_stats.playcount_std,
	users_stats.replays_watched_std, users_stats.total_hits_std, users_stats.level_std,
	users_stats.avg_accuracy_std, leaderboard_std.position as std_position,
	
	users_stats.ranked_score_taiko, users_stats.total_score_taiko, users_stats.playcount_taiko,
	users_stats.replays_watched_taiko, users_stats.total_hits_taiko, users_stats.level_taiko,
	users_stats.avg_accuracy_taiko, leaderboard_taiko.position as taiko_position,

	users_stats.ranked_score_ctb, users_stats.total_score_ctb, users_stats.playcount_ctb,
	users_stats.replays_watched_ctb, users_stats.total_hits_ctb, users_stats.level_ctb,
	users_stats.avg_accuracy_ctb, leaderboard_ctb.position as ctb_position,

	users_stats.ranked_score_mania, users_stats.total_score_mania, users_stats.playcount_mania,
	users_stats.replays_watched_mania, users_stats.total_hits_mania, users_stats.level_mania,
	users_stats.avg_accuracy_mania, leaderboard_mania.position as mania_position

FROM users
LEFT JOIN users_stats
ON users.id=users_stats.id
LEFT JOIN leaderboard_std
ON users.id=leaderboard_std.user
LEFT JOIN leaderboard_taiko
ON users.id=leaderboard_taiko.user
LEFT JOIN leaderboard_ctb
ON users.id=leaderboard_ctb.user
LEFT JOIN leaderboard_mania
ON users.id=leaderboard_mania.user
WHERE users.id=? AND users.allowed = '1'
LIMIT 1
`
	// Fuck.
	r := userFullResponse{}
	var (
		badges         string
		country        string
		showCountry    bool
		registeredOn   int64
		latestActivity int64
	)
	err := md.DB.QueryRow(query, md.C.Param("id")).Scan(
		&r.ID, &r.Username, &registeredOn, &r.Rank, &latestActivity,

		&r.UsernameAKA, &badges, &country, &showCountry,
		&r.PlayStyle, &r.FavouriteMode,

		&r.STD.RankedScore, &r.STD.TotalScore, &r.STD.PlayCount,
		&r.STD.ReplaysWatched, &r.STD.TotalHits, &r.STD.Level,
		&r.STD.Accuracy, &r.STD.GlobalLeaderboardRank,

		&r.Taiko.RankedScore, &r.Taiko.TotalScore, &r.Taiko.PlayCount,
		&r.Taiko.ReplaysWatched, &r.Taiko.TotalHits, &r.Taiko.Level,
		&r.Taiko.Accuracy, &r.Taiko.GlobalLeaderboardRank,

		&r.CTB.RankedScore, &r.CTB.TotalScore, &r.CTB.PlayCount,
		&r.CTB.ReplaysWatched, &r.CTB.TotalHits, &r.CTB.Level,
		&r.CTB.Accuracy, &r.CTB.GlobalLeaderboardRank,

		&r.Mania.RankedScore, &r.Mania.TotalScore, &r.Mania.PlayCount,
		&r.Mania.ReplaysWatched, &r.Mania.TotalHits, &r.Mania.Level,
		&r.Mania.Accuracy, &r.Mania.GlobalLeaderboardRank,
	)
	switch {
	case err == sql.ErrNoRows:
		return common.SimpleResponse(404, "That user could not be found!")
	case err != nil:
		md.Err(err)
		return Err500
	}

	r.Country = genCountry(md, r.ID, showCountry, country)
	r.Badges = badgesToArray(badges)

	r.RegisteredOn = time.Unix(registeredOn, 0)
	r.LatestActivity = time.Unix(latestActivity, 0)

	r.Code = 200
	return r
}

type userpageResponse struct {
	common.ResponseBase
	Userpage string `json:"userpage"`
}

// UserUserpageGET gets an user's userpage, as in the customisable thing.
func UserUserpageGET(md common.MethodData) common.CodeMessager {
	var r userpageResponse
	err := md.DB.QueryRow("SELECT userpage_content FROM users_stats WHERE id = ? LIMIT 1", md.C.Param("id")).Scan(&r.Userpage)
	switch {
	case err == sql.ErrNoRows:
		return common.SimpleResponse(404, "No user with that user ID!")
	case err != nil:
		md.Err(err)
		return Err500
	}
	r.Code = 200
	return r
}
