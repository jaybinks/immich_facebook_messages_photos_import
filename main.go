package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gabs "github.com/Jeffail/gabs/v2"
	"github.com/jaybinks/immich-go/browser"

	immich "github.com/jaybinks/immich-go/immich"
)

// var ImmichAPI *immich.ImmichClient
var ctx context.Context
var Albums map[string]string

var base_path string
var chatname []string

// Consts are in secrets.go (not in git)
/*
const ImmichURI = "http://192.168.0.100:2283/"
const ImmichAPIKey = "YOURKEYHERE"

base_path = "/Users/you/Downloads/your_activity_across_facebook/messages/inbox"

chatname = append(chatname, "melissachats_2592319644154807")
chatname = append(chatname, "chat2_3962147207173064")
*/

func main() {

	ImmichAPI, err := immich.NewImmichClient(ImmichURI, ImmichAPIKey, false)
	if err != nil {
	}

	ImmichAPI.SetEndPoint(ImmichURI + "api")

	ctx, _ = context.WithCancel(context.Background())

	err = ImmichAPI.PingServer(ctx)
	if err != nil {
	}

	// Get Album list
	AlbumList, _ := ImmichAPI.GetAllAlbums(ctx)
	Albums = make(map[string]string)
	for _, ThisAlbum := range AlbumList {
		Albums[ThisAlbum.AlbumName] = ThisAlbum.ID
	}

	for _, this_chat := range chatname {
		chat_path := fmt.Sprintf("%s/%s/", base_path, this_chat)
		process_facebook_chat(ImmichAPI, chat_path)
	}
}

func process_facebook_chat(ImmichAPI *immich.ImmichClient, chat_path string) {

	base_path := filepath.Base(chat_path)

	json, err := ioutil.ReadFile(chat_path + "message_1.json") //read the content of file
	if err != nil {
		fmt.Println(err)
		return
	}

	jsonParsed, err := gabs.ParseJSON([]byte(json))
	if err != nil {
		panic(err)
	}

	// Create Album name
	names := ""
	for _, child := range jsonParsed.S("participants").Children() {
		names += child.Path("name").Data().(string) + " "
	}
	AlbumName := "Facebook messages : " + names
	AlbumID := CreateAlbum(ImmichAPI, AlbumName)
	fmt.Printf("Uploading to Album : %s\n", AlbumName)

	for _, child := range jsonParsed.S("messages").Children() {

		var content string
		if child.Path("content").Data() != nil {
			content = child.Path("content").Data().(string)
		}

		sendername := child.Path("sender_name").Data().(string)

		var url string
		var timestamp string
		var AssetTime time.Time

		if child.Path("photos").Data() != nil {

			if child.Path("photos.0.uri").Data() != nil {
				url = child.Path("photos.0.uri").Data().(string)
				timestamp = child.Path("photos.0.creation_timestamp").String()
				timeint, err := strconv.ParseInt(timestamp, 10, 64)
				if err != nil {
				}
				AssetTime = time.Unix(timeint, 0)

				// Make path releative to our JSON
				pos := strings.Index(url, base_path) + len(base_path) + 1 // +1 for the leading "/"
				relpath := url[pos:]

				AssetID, err := upload(ImmichAPI, chat_path+relpath, AssetTime, sendername) //+"\n"+content)
				if err == nil {
					ImmichAPI.AddAssetToAlbum(ctx, AlbumID, []string{AssetID})
				}

			}
		} else if child.Path("videos").Data() != nil {
			var url string
			if child.Path("videos.0.uri").Data() != nil {
				url = child.Path("videos.0.uri").Data().(string)
				timestamp = child.Path("videos.0.creation_timestamp").String()
				timeint, err := strconv.ParseInt(timestamp, 10, 64)
				if err != nil {
				}
				AssetTime = time.Unix(timeint, 0)

				// Make path releative to our JSON
				pos := strings.Index(url, base_path) + len(base_path) + 1 // +1 for the leading "/"
				relpath := url[pos:]

				AssetID, err := upload(ImmichAPI, chat_path+relpath, AssetTime, sendername+"\n"+content)
				if err == nil {
					ImmichAPI.AddAssetToAlbum(ctx, AlbumID, []string{AssetID})
				}
			}

		}

		/*if url != "" {

			fmt.Printf("%s %s %s %s\n",
				AssetTime.String(),
				child.Path("sender_name").Data().(string),
				url,
				content)
		}*/
	}

}
func CreateAlbum(ImmichAPI *immich.ImmichClient, name string) string {
	id, exists := Albums[name]
	if exists {
		return id
	}

	alb, _ := ImmichAPI.CreateAlbum(ctx, name, nil)
	return alb.ID
}

func upload(ImmichAPI *immich.ImmichClient, File string, time time.Time, Description string) (string, error) {

	var la browser.LocalAssetFile
	la.FSys = os.DirFS("/")
	la.FileName = File[1:] // strip leading / for above
	la.DateTaken = time
	la.Favorite = false
	la.Title = filepath.Base(File)
	la.Description = Description
	//	fmt.Printf("la: %v\n", la)

	resp, err := ImmichAPI.AssetUpload(ctx, &la)

	if resp.Duplicate {

		/*
			Asset, _ := ImmichAPI.GetAssetByID(ctx, resp.ID)

			fmt.Printf("Duplicate : %s %s\n", resp.Duplicate, resp.ID)
			ImmichAPI.UpdateAsset(ctx, resp.ID, Asset)

			// Cant figure out API calls to set the date !!!!!
		*/
	}

	if err != nil {
		fmt.Printf("Error : %s\n", err.Error())
	} else {
		fmt.Printf("SUCCESS")
		return resp.ID, err
	}

	return "", err
}
