package handlers

import (
	"fmt"

	"net/http"

	"github.com/gorilla/mux"
	"github.com/vegardvb/cesium-terrain-server/log"
	"github.com/vegardvb/cesium-terrain-server/stores"
)

// An HTTP handler which returns a terrain tile resource
func TerrainHandler(store stores.Storer) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			t   stores.Terrain
			err error
		)

		defer func() {
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				log.Err(err.Error())
			}
		}()

		// get the tile coordinate from the URL
		vars := mux.Vars(r)
		err = t.ParseCoord(vars["x"], vars["y"], vars["z"])
		if err != nil {
			return
		}

		// Try and get a tile from the store
		err = store.Tile(vars["tileset"], &t)
		if err == stores.ErrNoItem {
			if store.TilesetStatus(vars["tileset"]) == stores.NOT_FOUND {
				err = nil
				http.Error(w,
					fmt.Errorf("the tileset `%s` does not exist", vars["tileset"]).Error(),
					http.StatusNotFound)
				return
			}

		} else if err != nil {
			return
		}

		body, err := t.MarshalBinary()
		if err != nil {
			return
		}

		// send the tile to the client
		headers := w.Header()
		headers.Set("Content-Type", "application/octet-stream")
		headers.Set("Content-Encoding", "gzip")
		headers.Set("Content-Disposition", "attachment;filename="+vars["y"]+".terrain")
		w.Write(body)
	}
}
