package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type seriesModel struct {
	Name       string         `json:"name" bson:"name"`
	ID         string         `json:"id" bson:"_id,omitempty"`
	OriginName string         `json:"origin_name" bson:"origin_name"`
	Year       string         `json:"year" bson:"year"`
	First      string         `json:"first" bson:"first"`
	Tags       []string       `json:"tags" bson:"tags"`
	Region     string         `json:"region" bson:"region"`
	Actors     []string       `json:"actors" bson:"actors"`
	Desc       string         `json:"desc" bson:"desc"`
	End        bool           `json:"end" bson:"end"`
	Videos     map[string]int `json:"videos" bson:"videos"`
	View       int            `json:"view" bson:"view"`
}
type outSeriesModel struct {
	Name       string         `json:"name" bson:"name"`
	ID         string         `json:"id" bson:"_id,omitempty"`
	OriginName string         `json:"origin_name" bson:"origin_name"`
	Region     string         `json:"region" bson:"region"`
	Actors     []string       `json:"actors" bson:"actors"`
	End        bool           `json:"end" bson:"end"`
	Videos     map[string]int `json:"videos" bson:"videos"`
	Year       string         `json:"year" bson:"year"`
	Last       int64          `json:"last" bson:"last"`
	View       int            `json:"view" bson:"view"`
}
type reqSModel struct {
	Region string `form:"region"`
	Search string `form:"search"`
}

func getSeries(c *gin.Context) {
	var params reqSModel
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "msg": err.Error()})
		return
	}

	log.Println(params)
	pipelines := bson.D{}
	if (reqSModel{}) == params {
		pipelines = append(pipelines, bson.E{"last", bson.D{{"$gt", time.Now().Unix() - 31*24*60*60}}})
	}
	if len(params.Region) != 0 {
		pipelines = append(pipelines, bson.E{"region", params.Region})
	}
	if len(params.Search) != 0 {
		pipelines = append(pipelines, bson.E{"$or",
			bson.A{
				bson.D{{"name", primitive.Regex{Pattern: params.Search, Options: "i"}}},
				bson.D{{"origin_name", primitive.Regex{Pattern: params.Search, Options: "i"}}},
				bson.D{{"actors", primitive.Regex{Pattern: params.Search, Options: "i"}}},
			}})
	}
	var items []outSeriesModel
	cur, err := dataBase.Collection("series").Find(context.Background(), pipelines)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "msg": err.Error()})
		return
	}
	for cur.Next(context.Background()) {
		var tmp outSeriesModel
		cur.Decode(&tmp)
		items = append(items, tmp)
	}
	c.JSON(http.StatusOK, items)
}
func getSone(c *gin.Context) {
	id, _ := c.Params.Get("id")
	var re seriesModel
	if err := dataBase.Collection("series").FindOneAndUpdate(context.Background(), bson.D{{"_id", id}}, bson.D{{"$inc", bson.D{{"view", 1}}}}).Decode(&re); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, re)
}
