package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type itemModel struct {
	Info     string                  `form:"info" json:"info" bson:"info" binding:"required"`
	Date     int64                   `json:"date" bson:"date"`
	Contact  string                  `form:"contact_info" json:"contact_info" bson:"contact" binding:"required"`
	Camp     string                  `form:"camp" json:"camp" bson:"camp" binding:"required"`
	Imgs     []string                `json:"imgs" bson:"imgs"`
	Wanted   int64                   `json:"wanted" bson:"wanted"`
	ID       primitive.ObjectID      `json:"id" bson:"_id,omitempty"`
	Solved   bool                    `json:"solved" bson:"solved"`
	Password string                  `form:"password" bson:"password" binding:"required" json:"-"`
	ImgsUp   []*multipart.FileHeader `form:"imgs_up" bson:"-" json:"-"`
	Thbs     []*multipart.FileHeader `form:"thbs" bson:"-" json:"-"`
}
type reqModel struct {
	Search string `form:"search"`
	Region string `form:"region"`
	Size   int64  `form:"size" binding:"required"`
	Start  string `form:"fi" binding:"required"`
}
type newreqModel struct {
	Search string `form:"search"`
	Region string `form:"region"`
	Start  string `form:"fi" binding:"required"`
}
type wantModel struct {
	ID primitive.ObjectID `json:"id" bson:"_id" binding:"required"`
}
type markModel struct {
	ID       primitive.ObjectID `json:"id" bson:"_id" binding:"required"`
	Password string             `json:"password" bson:"password" binding:"required"`
}

func addItem(c *gin.Context) {
	var item itemModel
	if err := c.ShouldBind(&item); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "msg": err.Error()})
		return
	}
	item.Date = time.Now().Unix()
	if item.Password == "無力なスペース" {
		item.Camp = "系统广播"
	}
	item.Imgs = []string{}
	for index, file := range item.ImgsUp {
		f, _ := file.Open()
		defer f.Close()
		nameMd5 := md5.New()
		if _, err := io.Copy(nameMd5, f); err != nil {
			log.Fatal(err)
		}
		name := hex.EncodeToString(nameMd5.Sum(nil))
		if err := c.SaveUploadedFile(file, globalConf.ResDir+"/imgs/"+name); err != nil {
			c.JSON(http.StatusOK, gin.H{"status": false, "msg": err.Error()})
			return
		}
		if err := c.SaveUploadedFile(item.Thbs[index], globalConf.ResDir+"/imgs/"+name+"_thb.jpeg"); err != nil {
			c.JSON(http.StatusOK, gin.H{"status": false, "msg": err.Error()})
			return
		}
		item.Imgs = append(item.Imgs, name)
	}
	re, err := dataBase.Collection("posts").InsertOne(context.Background(), item)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "id": re.InsertedID, "msg": "发布成功"})
}
func findOne(c *gin.Context) {
	val, _ := c.Params.Get("id")
	id, err := primitive.ObjectIDFromHex(val)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "msg": "id错误"})
		return
	}
	var re []itemModel
	var tmp itemModel
	if err := dataBase.Collection("posts").FindOne(context.Background(), bson.D{{"_id", id}, {"solved", false}}).Decode(&tmp); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "msg": err.Error()})
		return
	}
	re = append(re, tmp)
	c.JSON(http.StatusOK, re)
}
func latestItems(c *gin.Context) {
	var params reqModel
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "msg": err.Error()})
		return
	}
	id, err := primitive.ObjectIDFromHex(params.Start)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "msg": "id错误"})
		return
	}
	var re []itemModel
	foptions := options.Find()
	foptions.SetSort(bson.D{{"_id", -1}})
	foptions.SetLimit(params.Size)
	// foptions.SetProjection(bson.D{{"password", 0}})

	pipelines := bson.D{{"info", primitive.Regex{Pattern: params.Search, Options: "i"}}, {"solved", false}, {"_id", bson.D{{"$lt", id}}}}
	if len(params.Region) != 0 {
		pipelines = append(pipelines, bson.E{"camp", bson.D{{"$in", bson.A{params.Region, "系统广播"}}}})
	}
	cur, err := dataBase.Collection("posts").Find(context.Background(), pipelines, foptions)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "msg": err.Error()})
		return
	}
	for cur.Next(context.Background()) {
		var tmp itemModel
		cur.Decode(&tmp)
		re = append(re, tmp)
	}
	c.JSON(http.StatusOK, re)
}

func markFlag(c *gin.Context) {
	var req markModel
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "msg": err.Error()})
		return
	}
	re := dataBase.Collection("posts").FindOneAndUpdate(context.Background(), bson.D{
		{"_id", req.ID},
		{"password", req.Password},
	}, bson.D{
		{"$set", bson.D{
			{"solved", true},
		}},
	})
	if re.Err() != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "msg": "密码错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "msg": "标记成功"})
}

func wanted(c *gin.Context) {
	var req wantModel
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "msg": err.Error()})
		return
	}
	if re := dataBase.Collection("posts").FindOneAndUpdate(context.Background(), bson.D{{"_id", req.ID}}, bson.D{{"$inc", bson.D{{"wanted", 1}}}}); re.Err() != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "msg": re.Err().Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true})
}

func anynew(c *gin.Context) {
	var params newreqModel
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "msg": err.Error()})
		return
	}
	id, err := primitive.ObjectIDFromHex(params.Start)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "msg": "id错误"})
		return
	}
	pipelines := bson.D{{"info", primitive.Regex{Pattern: params.Search, Options: "i"}}, {"solved", false}, {"_id", bson.D{{"$gt", id}}}}
	if len(params.Region) != 0 {
		pipelines = append(pipelines, bson.E{"camp", bson.D{{"$in", bson.A{params.Region, "系统广播"}}}})
	}
	re := dataBase.Collection("posts").FindOne(context.Background(), pipelines)
	c.JSON(http.StatusOK, re.Err() == nil)
}
