"use strict";

/*
 * A class for storing global state required by the application.
 */
function Globals() {
	this.cgi = '/cgi-bin/locviz';
	this.mimeDefault = 'application/x-www-form-urlencoded';
	this.tileSize = 256.0;
}

/*
 * The global state object.
 */
const globals = new Globals();

/*
 * A class implementing data storage.
 */
function Storage() {
	const g_map = new WeakMap();

	/*
	 * Store a value under a key inside an element.
	 */
	this.put = function(elem, key, value) {
		let map = g_map.get(elem);

		/*
		 * Check if element is still unknown.
		 */
		if (map == null) {
			map = new Map();
			g_map.set(elem, map);
		}

		map.set(key, value);
	};

	/*
	 * Fetch a value from a key inside an element.
	 */
	this.get = function(elem, key, value) {
		const map = g_map.get(elem);

		/*
		 * Check if element is unknown.
		 */
		if (map == null) {
			return null;
		} else {
			const value = map.get(key);
			return value;
		}

	};

	/*
	 * Check if a certain key exists inside an element.
	 */
	this.has = function(elem, key) {
		const map = g_map.get(elem);

		/*
		 * Check if element is unknown.
		 */
		if (map == null) {
			return false;
		} else {
			const value = map.has(key);
			return value;
		}

	};

	/*
	 * Remove a certain key from an element.
	 */
	this.remove = function(elem, key) {
		const map = g_map.get(elem);

		/*
		 * Check if element is known.
		 */
		if (map != null) {
			map.delete(key);

			/*
			 * If inner map is now empty, remove it from outer map.
			 */
			if (map.size === 0) {
				g_map.delete(elem);
			}

		}

	};

}

/*
 * The global storage object.
 */
const storage = new Storage();

/*
 * A class supposed to make life a little easier.
 */
function Helper() {

	/*
	 * Blocks or unblocks the site for user interactions.
	 */
	this.blockSite = function(blocked) {
		const blocker = document.getElementById('blocker');
		let displayStyle = '';

		/*
		 * If we should block the site, display blocker, otherwise hide it.
		 */
		if (blocked) {
			displayStyle = 'block';
		} else {
			displayStyle = 'none';
		}

		/*
		 * Apply style if the site has a blocker.
		 */
		if (blocker != null) {
			blocker.style.display = displayStyle;
		}

	};

	/*
	 * Clean a (string) value obtained from a date input element.
	 *
	 * Remove whitespace and replace the empty string by null value.
	 */
	this.cleanValue = function(v) {

		/*
		 * Null values are not handled.
		 */
		if (v == null) {
			return null;
		} else {
			v = v.toString();
			v = v.trim();

			/*
			 * Replace empty string by null.
			 */
			if (v == "") {
				return null;
			} else {
				return v;
			}

		}

	}

	/*
	 * Removes all child nodes from an element.
	 */
	this.clearElement = function(elem) {

		/*
		 * As long as the element has child nodes, remove one.
		 */
		while (elem.hasChildNodes()) {
			const child = elem.firstChild;
			elem.removeChild(child);
		}

	};

	/*
	 * Parse JSON string into an object without raising exceptions.
	 */
	this.parseJSON = function(jsonString) {

		/*
		 * Try to parse JSON structure.
		 */
		try {
			const obj = JSON.parse(jsonString);
			return obj;
		} catch (ex) {
			return null;
		}

	};

	/*
	 * Convert fractional degrees to degrees, minutes, seconds.
	 */
	this.convertToDMS = function(dd, suffixPos, suffixNeg) {
		const deg = dd | 0;
		const degString = deg.toFixed(0);
		const degAbs = Math.abs(deg);
		const degAbsString = degAbs.toFixed(0);
		const frac = Math.abs(dd - deg);
		const m = (frac * 60) | 0;
		const mString = m.toFixed(0);
		const sec = (frac * 3600) - (m * 60);
		const secString = sec.toFixed(2);
		let result = '';

		/*
		 * Check whether to use sign or suffix.
		 */
		if ((suffixPos != null) & (suffixNeg != null)) {
			let suffix = '';

			/*
			 * Choose suffix.
			 */
			if (deg >= 0) {
				suffix = ' ' + suffixPos;
			} else {
				suffix = ' ' + suffixNeg;
			}

			result = degAbsString + '° ' + mString + '\' ' + secString + '"' + suffix;
		} else {
			result = degString + '° ' + mString + '\' ' + secString + '"';
		}

		return result;
	}

	/*
	 * Transform Mercator coordinates into geographic coordinates.
	 */
	this.transformCoordinates = function(xpos, ypos) {
		const twoPi = 2.0 * Math.PI;
		const halfPi = 0.5 * Math.PI;
		const longitude = 360.0 * xpos;
		const longitudeDMS = this.convertToDMS(longitude, 'E', 'W');
		const ya = twoPi * ypos;
		const yb = Math.exp(ya);
		const yc = Math.atan(yb);
		const yd = 2.0 * yc;
		const ye = yd - halfPi;
		const latitude = (360.0 / twoPi) * ye;
		const latitudeDMS = this.convertToDMS(latitude, 'N', 'S');

		/*
		 * This is the result.
		 */
		const result = {
			longitude: longitudeDMS,
			latitude: latitudeDMS
		};

		return result;
	};

}

/*
 * The (global) helper object.
 */
const helper = new Helper();

/*
 * A class implementing an Ajax engine.
 */
function Ajax() {

	/*
	 * Sends an Ajax request to the server.
	 *
	 * Parameters:
	 * - method (string): The request method (e. g. 'GET', 'POST', ...).
	 * - url (string): The request URL.
	 * - data (string): Data to be passed along the request (e. g. form data).
	 * - mimeType (string): Content type (MIME type) of the content sent to the server.
	 * - callback (function): The function to be called when a response is
	 *	returned from the server.
	 * - block (boolean): Whether the site should be blocked.
	 *
	 * Returns: Nothing.
	 */
	this.request = function(method, url, data, mimeType, callback, block) {
		const xhr = new XMLHttpRequest();

		/*
		 * Event handler for ReadyStateChange event.
		 */
		xhr.onreadystatechange = function() {
			helper.blockSite(block);

			/*
			 * If we got a response, pass the response text to
			 * the callback function.
			 */
			if (this.readyState === 4) {

				/*
				 * If we blocked the site on the request,
				 * unblock it on the response.
				 */
				if (block) {
					helper.blockSite(false);
				}

				/*
				 * Check if callback is registered.
				 */
				if (callback !== null) {
					const content = xhr.responseText;
					callback(content);
				}

			}

		};

		xhr.open(method, url, true);

		/*
		 * Set MIME type if requested.
		 */
		if (mimeType !== null) {
			xhr.setRequestHeader('Content-type', mimeType);
		}

		xhr.send(data);
	};

	/*
	 * Requests an image from the server.
	 *
	 * Parameters:
	 * - url (string): The request URL.
	 * - data (string): Data to be passed along the request (e. g. form data).
	 * - callback (function): The function to be called when a response is
	 *	returned from the server.
	 * - block (boolean): Whether the site should be blocked.
	 * - id (integer): The id for this image request.
	 *
	 * Returns: Nothing.
	 */
	this.requestImage = function(url, data, callback, block, id) {
		helper.blockSite(block);
		const img = new Image();

		/*
		 * Event handler for load event.
		 */
		const eventSuccess = function() {

			/*
			 * If we blocked the site on the request,
			 * unblock it on the response.
			 */
			if (block) {
				helper.blockSite(false);
			}

			/*
			 * Check if callback is registered.
			 */
			if (callback !== null) {
				callback(img, id);
			}

		};

		/*
		 * Event handler for error event.
		 */
		const eventError = function() {

			/*
			 * If we blocked the site on the request,
			 * unblock it on the response.
			 */
			if (block) {
				helper.blockSite(false);
			}

			/*
			 * Check if callback is registered.
			 */
			if (callback !== null) {
				callback(null, id);
			}

		};

		img.onload = eventSuccess;
		img.onerror = eventError;
		const uri = url + '?' + data;
		img.src = uri;
	};

}

/*
 * The (global) Ajax engine.
 */
const ajax = new Ajax();

/*
 * A class implementing a key-value-pair.
 */
function KeyValuePair(key, value) {
	const g_key = key;
	const g_value = value;

	/*
	 * Returns the key stored in this key-value pair.
	 */
	this.getKey = function() {
		return g_key;
	};

	/*
	 * Returns the value stored in this key-value pair.
	 */
	this.getValue = function() {
		return g_value;
	};

}

/*
 * A class implementing a JSON request.
 */
function Request() {
	const g_keyValues = Array();

	/*
	 * Append a key-value-pair to a request.
	 */
	this.append = function(key, value) {
		const kv = new KeyValuePair(key, value);
		g_keyValues.push(kv);
	};

	/*
	 * Returns the URL encoded data for this request.
	 */
	this.getData = function() {
		const numPairs = g_keyValues.length;
		let s = '';

		/*
		 * Iterate over the key-value pairs.
		 */
		for (let i = 0; i < numPairs; i++) {
			const keyValue = g_keyValues[i];
			const key = keyValue.getKey();
			const keyEncoded = encodeURIComponent(key);
			const value = keyValue.getValue();
			const valueEncoded = encodeURIComponent(value);

			/*
			 * If this is not the first key-value pair, we need a separator.
			 */
			if (i > 0) {
				s += '&';
			}

			s += keyEncoded + '=' + valueEncoded;
		}

		return s;
	};

}

/*
 * This class implements helper functions to build a user interface.
 */
function UI() {
	const self = this;

	/*
	 * Creates a generic UI element container with a label.
	 */
	this.createElement = function(labelCaption) {
		const labelDiv = document.createElement('div');
		labelDiv.className = 'label';
		const labelNode = document.createTextNode(labelCaption);
		labelDiv.appendChild(labelNode)
		const uiElement = document.createElement('div');
		uiElement.className = 'uielement';
		uiElement.appendChild(labelDiv);
		return uiElement;
	};

	/*
	 * Calculate the IDs and positions of the tiles required to
	 * display a certain portion of the map and their positions
	 * inside the coordinate system.
	 */
	this.calculateTiles = function(xres, yres, zoom, xpos, ypos) {
		const zoomExp = 0.2 * zoom;
		const zoomFac = Math.pow(2.0, zoomExp);
		const zoomFacInv = 1.0 / zoomFac;
		const halfWidth = 0.5 * zoomFacInv;
		const aspectRatio = yres / xres;
		const halfHeight = aspectRatio * halfWidth;
		const minX = xpos - halfWidth;
		const maxX = xpos + halfWidth;
		const minY = ypos - halfHeight;
		const maxY = ypos + halfHeight;
		const tileSize = globals.tileSize;
		const osmZoomFloat = Math.log2(zoomFac * (xres / tileSize))
		let osmZoom = Math.floor(osmZoomFloat);

		/*
		 * Limit OSM zoom.
		 */
		if (osmZoom < 0.0) {
			osmZoom = 0.0;
		} else if (osmZoom > 19.0) {
			osmZoom = 19.0;
		}

		const maxTileId = (1 << osmZoom) - 1;
		const dxPerTile = Math.pow(2.0, -osmZoom);
		const idxMinX = Math.floor((minX + 0.5) / dxPerTile);
		const idxMaxX = Math.floor((maxX + 0.5) / dxPerTile);
		const idxMinY = Math.floor((0.5 - maxY) / dxPerTile);
		const idxMaxY = Math.floor((0.5 - minY) / dxPerTile);
		const tileDescriptors = [];

		/*
		 * Iterate over the Y axis.
		 */
		for (let idxY = idxMinY; idxY <= idxMaxY; idxY++) {

			/*
			 * Iterate over the X axis.
			 */
			for (let idxX = idxMinX; idxX <= idxMaxX; idxX++) {

				/*
				 * Check if tile ID is valid.
				 */
				if ((idxX >= 0) & (idxX <= maxTileId) & (idxY >= 0) & (idxY <= maxTileId)) {
					const tileMinX = (idxX * dxPerTile) - 0.5;
					const tileMaxX = tileMinX + dxPerTile;
					const tileMaxY = 0.5 - (idxY * dxPerTile);
					const tileMinY = tileMaxY - dxPerTile;

					/*
					 * Calculate tile IDs and limits.
					 *
					 * OSM coordinates have X axis to the right and
					 * Y axis downwards.
					 *
					 * Our coordinates have X axis to the right and
					 * Y axis upwards and interval [-0.5, 0.5].
					 */
					const tileDescriptor = {
						osmX: idxX,
						osmY: idxY,
						osmZoom: osmZoom,
						dx: dxPerTile,
						minX: tileMinX,
						maxX: tileMaxX,
						minY: tileMinY,
						maxY: tileMaxY,
						imgData: null,
						fetched: false
					};

					tileDescriptors.push(tileDescriptor);
				}

			}

		}

		return tileDescriptors;
	};

	/*
	 * Fetches a of tiles from the server and notifies listener
	 * about update.
	 */
	this.fetchTile = function(tileDescriptor, listener) {
		const x = tileDescriptor.osmX;
		const y = tileDescriptor.osmY;
		const z = tileDescriptor.osmZoom;
		const rq = new Request();
		rq.append('cgi', 'get-tile');
		const xString = x.toString();
		rq.append('x', xString);
		const yString = y.toString();
		rq.append('y', yString);
		const zString = z.toString();
		rq.append('z', zString);
		const cgi = globals.cgi;
		const data = rq.getData();

		/*
		 * This is called when the server sends data.
		 */
		const callback = function(img, idResponse) {
			listener(tileDescriptor, img);
		};

		ajax.requestImage(cgi, data, callback, false, null);
	};

	/*
	 * Draws a tile on the map canvas.
	 */
	this.drawTile = function(tileDescriptor) {
		const img = tileDescriptor.imgData;

		/*
		 * Check if current tile has image data attached.
		 */
		if (img !== null) {
			const cvs = document.getElementById('map_canvas');
			const xres = cvs.scrollWidth;
			const yres = cvs.scrollHeight;
			const posX = storage.get(cvs, 'posX');
			const posY = storage.get(cvs, 'posY');
			const zoom = storage.get(cvs, 'zoomLevel');
			const zoomExp = 0.2 * zoom;
			const zoomFac = Math.pow(2.0, zoomExp);
			const zoomFacInv = 1.0 / zoomFac;
			const halfWidth = 0.5 * zoomFacInv;
			const aspectRatio = yres / xres;
			const halfHeight = aspectRatio * halfWidth;
			const minX = posX - halfWidth;
			// const maxX = posX + halfWidth;
			// const minY = posY - halfHeight;
			const maxY = posY + halfHeight;
			const tileMinX = tileDescriptor.minX;
			const tileMinY = tileDescriptor.minY;
			const tileMaxX = tileDescriptor.maxX;
			const tileMaxY = tileDescriptor.maxY;
			const destX = xres * ((tileMinX - minX) * zoomFac);
			const destY = xres * ((maxY - tileMaxY) * zoomFac);
			const destWidth = xres * ((tileMaxX - tileMinX) * zoomFac);
			const destHeight = xres * ((tileMaxY - tileMinY) * zoomFac);
			const ctx = cvs.getContext('2d');
			ctx.drawImage(img, destX, destY, destWidth, destHeight);
		}

	};

	/*
	 * This is called when a new map tile has been fetched.
	 */
	this.updateTiles = function(tileDescriptors) {
		const cvs = document.getElementById('map_canvas');
		const width = cvs.scrollWidth;
		const height = cvs.scrollHeight;
		const ctx = cvs.getContext('2d');
		ctx.clearRect(0, 0, width, height);
		const tiles = storage.get(cvs, 'osmTiles');

		/*
		 * Check if map tiles have to be drawn.
		 */
		if (tiles !== null) {
			const numTiles = tiles.length;

			/*
			 * Draw every single map tile.
			 */
			for (let i = 0; i < numTiles; i++) {
				const currentTile = tiles[i];

				/*
				 * Draw tile if map tile is available.
				 */
				if (currentTile !== null) {
					this.drawTile(currentTile);
				}

			}

		}

		const img = storage.get(cvs, 'lastImage');

		/*
		 * Check if image overlay has to be drawn.
		 */
		if (img !== null) {
			ctx.drawImage(img, 0, 0);
		}

	};

	/*
	 * Fetch a list of map tiles concurrently and invoke callback on each change.
	 */
	this.fetchTiles = function(tileIds, callback) {

		/*
		 * Internal callback invoked by fetchTile(...).
		 */
		const internalCallback = function(tileDescriptor, img) {
			tileDescriptor.imgData = img;
			tileDescriptor.fetched = true;
			callback(tileIds);
		};

		/*
		 * Iterate over all tiles and fetch them.
		 */
		for (let i = 0; i < tileIds.length; i++) {
			const currentTile = tileIds[i];
			const fetched = currentTile.fetched;

			/*
			 * Check if we have to fetch this tile.
			 */
			if (fetched === false) {
				self.fetchTile(currentTile, internalCallback);
			}

		}

	};

	/*
	 * Redraw the map with the same image, but a different offset.
	 */
	this.moveMap = function(xoffs, yoffs) {
		const cvs = document.getElementById('map_canvas');
		const img = storage.get(cvs, 'lastImage');

		/*
		 * Load or store x-offset.
		 */
		if (xoffs !== null) {
			storage.put(cvs, 'offsetX', xoffs);
		} else {
			xoffs = storage.get(cvs, 'offsetX');
		}

		/*
		 * Load or store y-offset.
		 */
		if (yoffs !== null) {
			storage.put(cvs, 'offsetY', yoffs);
		} else {
			yoffs = storage.get(cvs, 'offsetY');
		}

		/*
		 * Check if there is a stored image.
		 */
		if (img !== null) {
			const width = cvs.scrollWidth;
			const height = cvs.scrollHeight;
			const zoomLevelRequested = storage.get(cvs, 'zoomLevel');
			const zoomLevelImage = storage.get(cvs, 'imageZoom');
			const zoomLevelDiff = zoomLevelRequested - zoomLevelImage;
			const zoomFac = Math.pow(2.0, 0.2 * zoomLevelDiff);
			const scaledWidth = width * zoomFac;
			const scaledHeight = height * zoomFac;
			const ctx = cvs.getContext('2d');
			ctx.clearRect(0, 0, width, height);
			const dx = xoffs + (0.5 * (width - scaledWidth));
			const dy = yoffs + (0.5 * (height - scaledHeight));
			ctx.drawImage(img, 0, 0, width, height, dx, dy, scaledWidth, scaledHeight);
		}

	};

	/*
	 * Updates the image element with a new view of the map.
	 */
	this.updateMap = function(xres, yres, xpos, ypos, zoom, mintime, maxtime, useOSMTiles) {
		/* Earth circumference at the equator. */
		const circ = 40074;
		const rq = new Request();
		rq.append('cgi', 'render');
		const xresString = xres.toString();
		rq.append('xres', xresString);
		const yresString = yres.toString();
		rq.append('yres', yresString);
		const xposString = xpos.toString();
		rq.append('xpos', xposString);
		const yposString = ypos.toString();
		rq.append('ypos', yposString);
		const zoomString = zoom.toString();
		rq.append('zoom', zoomString);
		const northingField = document.getElementById('northing_field');
		northingField.value = ypos.toFixed(10);
		const eastingField = document.getElementById('easting_field');
		eastingField.value = xpos.toFixed(10);
		const zoomField = document.getElementById('zoom_field');
		zoomField.value = zoomString;
		const xposKM = xpos * circ;
		const yposKM = ypos * circ;
		const eastingFieldKM = document.getElementById('easting_field_km');
		eastingFieldKM.value = xposKM.toFixed(3);
		const northingFieldKM = document.getElementById('northing_field_km');
		northingFieldKM.value = yposKM.toFixed(3);
		const longLat = helper.transformCoordinates(xpos, ypos);
		const longitude = longLat.longitude;
		const latitude = longLat.latitude;
		const longitudeField = document.getElementById('longitude_field');
		longitudeField.value = longitude;
		const latitudeField = document.getElementById('latitude_field');
		latitudeField.value = latitude;

		/*
		 * Use min-time.
		 */
		if (mintime !== null) {
			const mintimeString = mintime.toString();
			rq.append('mintime', mintimeString);
		}

		/*
		 * Use max-time.
		 */
		if (maxtime !== null) {
			const maxtimeString = maxtime.toString();
			rq.append('maxtime', maxtimeString);
		}

		rq.append('usebg', 'false');
		const cgi = globals.cgi;
		const data = rq.getData();
		const cvs = document.getElementById('map_canvas');
		const idRequest = storage.get(cvs, 'imageRequestId');
		const currentRequestId = idRequest + 1;
		storage.put(cvs, 'imageRequestId', currentRequestId);
		storage.put(cvs, 'osmTiles', []);

		/*
		 * This is called when the server sends data.
		 */
		const callback = function(img, idResponse) {
			const lastResponse = storage.get(cvs, 'imageResponseId');

			/*
			 * Check if the response is more current than what we display.
			 */
			if (idResponse > lastResponse) {
				storage.put(cvs, 'lastImage', img);
				storage.put(cvs, 'imageResponseId', idResponse);
				storage.put(cvs, 'imageZoom', zoom);
				storage.put(cvs, 'offsetX', 0);
				storage.put(cvs, 'offsetY', 0);
				storage.put(cvs, 'dragInterrupted', true);
				const width = cvs.scrollWidth;
				const height = cvs.scrollHeight;
				cvs.width = width;
				cvs.height = height;
				const ctx = cvs.getContext('2d');
				ctx.clearRect(0, 0, width, height);
				ctx.drawImage(img, 0, 0);

				/*
				 * Check if we should use OSM tiles.
				 */
				if (useOSMTiles) {
					const tileIds = self.calculateTiles(xres, yres, zoom, xpos, ypos);
					storage.put(cvs, 'osmTiles', tileIds);

					/*
					 * Internal callback necessary to have
					 * "this" reference.
					 */
					const internalCallback = function() {
						self.updateTiles();
					};

					self.fetchTiles(tileIds, internalCallback);
				}

			}

		};

		ajax.requestImage(cgi, data, callback, true, currentRequestId);
	};

}

const ui = new UI();

/*
 * This class implements all handler functions for user interaction.
 */
function Handler() {
	const self = this;
	this._timeoutScroll = null;
	this._timeoutResize = null;

	/*
	 * This is called when the map needs to be refreshed.
	 */
	this.refresh = function() {
		const cvs = document.getElementById('map_canvas');
		const width = cvs.scrollWidth;
		const height = cvs.scrollHeight;
		const posX = storage.get(cvs, 'posX');
		const posY = storage.get(cvs, 'posY');
		const zoom = storage.get(cvs, 'zoomLevel');
		const timeMin = storage.get(cvs, 'minTime');
		const timeMax = storage.get(cvs, 'maxTime');
		const useOSMTiles = storage.get(cvs, 'useOSMTiles');
		ui.updateMap(width, height, posX, posY, zoom, timeMin, timeMax, useOSMTiles);
	};

	/*
	 * Extracts coordinates from a single-point touch event.
	 */
	this.touchEventToCoordinates = function(e) {
		const cvs = e.target;
		const rect = cvs.getBoundingClientRect();
		const offsetX = rect.left;
		const offsetY = rect.top;
		const touches = e.targetTouches;
		const numTouches = touches.length;
		let touch = null;

		/*
		 * If there are touches, extract the first one.
		 */
		if (numTouches > 0) {
			touch = touches.item(0);
		}

		let x = 0.0;
		let y = 0.0;

		/*
		 * If a touch was extracted, calculate coordinates relative to
		 * the element position.
		 */
		if (touch !== null) {
			const touchX = touch.pageX;
			x = touchX - offsetX;
			const touchY = touch.pageY;
			y = touchY - offsetY;
		}

		const result = {
			x: x,
			y: y
		};

		return result;
	};

	/*
	 * Extracts the distance from a multi-point touch event.
	 */
	this.touchEventToDistance = function(e) {
		const touches = e.targetTouches;
		const numTouches = touches.length;

		/*
		 * If there are two touches, extract them and calculate their distance.
		 */
		if (numTouches === 2) {
			const touchA = touches.item(0);
			const touchB = touches.item(1);
			const xA = touchA.pageX;
			const yA = touchA.pageY;
			const xB = touchB.pageX;
			const yB = touchB.pageY;
			const dX = xB - xA;
			const dY = yB - yA;
			const dXSquared = dX * dX;
			const dYSquared = dY * dY;
			const dSquared = dXSquared + dYSquared;
			const distance = Math.sqrt(dSquared);
			return distance;
		} else {
			return 0.0;
		}

	};

	/*
	 * This is called when a user touches the map.
	 */
	this.touchStart = function(e) {
		e.preventDefault();
		const cvs = e.target;
		const touches = e.targetTouches;
		const numTouches = touches.length;
		const singleTouch = (numTouches === 1);
		const multiTouch = (numTouches > 1);

		/*
		 * Single touch moves the map, multi touch zooms.
		 */
		if (singleTouch) {
			const coords = self.touchEventToCoordinates(e);
			const x = coords.x;
			const y = coords.y;
			storage.put(cvs, 'mouseButton', true);
			storage.put(cvs, 'dragInterrupted', false);
			storage.put(cvs, 'touchStartX', x);
			storage.put(cvs, 'touchStartY', y);
			storage.put(cvs, 'touchStartDistance', 0.0);
			storage.put(cvs, 'touchLastX', x);
			storage.put(cvs, 'touchLastY', y);
		} else if (multiTouch) {
			const dist = self.touchEventToDistance(e);
			storage.put(cvs, 'mouseButton', false);
			storage.put(cvs, 'dragInterrupted', true);
			storage.put(cvs, 'touchStartX', 0);
			storage.put(cvs, 'touchStartY', 0);
			storage.put(cvs, 'touchStartDistance', dist);
			storage.put(cvs, 'touchLastX', 0);
			storage.put(cvs, 'touchLastY', 0);
		}

	};

	/*
	 * This is called when a user moves a finger on the map.
	 */
	this.touchMove = function(e) {
		const cvs = e.target;
		const btn = storage.get(cvs, 'mouseButton');

		/*
		 * If mouse button is depressed, move map, otherwise zoom map.
		 */
		if (btn) {
			const interrupted = storage.get(cvs, 'dragInterrupted');

			/*
			 * Only care if drag was not interrupted.
			 */
			if (!interrupted) {
				const touches = e.targetTouches;
				const numTouches = touches.length;
				const singleTouch = (numTouches === 1);

				/*
				 * Only process single touches, not multi-touch
				 * gestures.
				 */
				if (singleTouch) {
					e.preventDefault();
					const coords = self.touchEventToCoordinates(e);
					const x = coords.x;
					const y = coords.y;
					storage.put(cvs, 'touchLastX', x);
					storage.put(cvs, 'touchLastY', y);
					const startX = storage.get(cvs, 'touchStartX');
					const startY = storage.get(cvs, 'touchStartY');
					const relX = x - startX;
					const relY = y - startY;
					ui.moveMap(relX, relY);
				}

			} else {
				ui.moveMap(0, 0);
			}

		} else {
			const touches = e.targetTouches;
			const numTouches = touches.length;
			const multiTouch = (numTouches > 1);

			/*
			 * Only process multi-touch gestures, not single
			 * touches.
			 */
			if (multiTouch) {
				e.preventDefault();
				const dist = self.touchEventToDistance(e);
				const startDist = storage.get(cvs, 'touchStartDistance');
				const frac = dist / startDist;
				const diffZoom = Math.round(5.0 * (Math.log(frac) / Math.log(2.0)));
				let zoom = storage.get(cvs, 'imageZoom');
				zoom += diffZoom;

				/*
				 * Zoom level shall not go below zero.
				 */
				if (zoom < 0) {
					zoom = 0;
				}

				/*
				 * Zoom level shall not go above 120.
				 */
				if (zoom > 120) {
					zoom = 120;
				}

				storage.put(cvs, 'zoomLevel', zoom);
				ui.moveMap(null, null);
			}

		}

	};

	/*
	 * This is called when a user lifts a finger off the map.
	 */
	this.touchEnd = function(e) {
		const cvs = e.target;
		const interrupted = storage.get(cvs, 'dragInterrupted');

		/*
		 * Only move map position if drag was not interrupted.
		 */
		if (interrupted === true) {
			self.refresh();
		} else {
			const touches = e.targetTouches;
			const numTouches = touches.length;
			const noMoreTouches = (numTouches === 0);

			/*
			 * Only commit value after the last finger has
			 * been lifted off.
			 */
			if (noMoreTouches) {
				e.preventDefault();
				const x = storage.get(cvs, 'touchLastX');
				const y = storage.get(cvs, 'touchLastY');
				const startX = storage.get(cvs, 'touchStartX');
				const startY = storage.get(cvs, 'touchStartY');
				const relX = x - startX;
				const relY = y - startY;
				const zoom = storage.get(cvs, 'zoomLevel');
				const width = cvs.scrollWidth;
				const fracX = relX / width;
				const fracY = relY / width;
				const zoomFac = Math.pow(2.0, -0.2 * zoom);
				const moveX = zoomFac * fracX;
				const moveY = zoomFac * fracY;
				let posX = storage.get(cvs, 'posX');
				let posY = storage.get(cvs, 'posY');
				posX -= moveX;
				posY += moveY;
				storage.put(cvs, 'posX', posX);
				storage.put(cvs, 'posY', posY);
				self.refresh();
			}

		}

	};

	/*
	 * This is called when a user cancels a touch action.
	 */
	this.touchCancel = function(e) {
		const cvs = e.target;
		const btn = storage.get(cvs, 'mouseButton');

		/*
		 * Abort action if mouse button was depressed.
		 */
		if (btn === true) {
			storage.put(cvs, 'dragInterrupted', true);
			ui.moveMap(0, 0);
		}

		storage.put(cvs, 'mouseButton', false);
	};

	/*
	 * This is called when the user presses the mouse button.
	 */
	this.mouseDown = function(e) {
		const btn = e.button;

		/*
		 * Check if left mouse button was depressed.
		 */
		if (btn === 0) {
			const cvs = e.target;
			const x = e.offsetX;
			const y = e.offsetY;
			storage.put(cvs, 'mouseButton', true);
			storage.put(cvs, 'mouseStartX', x);
			storage.put(cvs, 'mouseStartY', y);
			storage.put(cvs, 'dragInterrupted', false);
		}

	};

	/*
	 * This is called when the user releases the mouse button.
	 */
	this.mouseUp = function(e) {
		const btn = e.button;

		/*
		 * Check if left mouse button was released.
		 */
		if (btn === 0) {
			const cvs = e.target;
			const x = e.offsetX;
			const y = e.offsetY;
			storage.put(cvs, 'mouseButton', false);
			const interrupted = storage.get(cvs, 'dragInterrupted');

			/*
			 * Load a new position if drag was not interrupted.
			 */
			if (interrupted === true) {
				ui.moveMap(0, 0);
			} else {
				const startX = storage.get(cvs, 'mouseStartX');
				const startY = storage.get(cvs, 'mouseStartY');
				const relX = x - startX;
				const relY = y - startY;
				const zoom = storage.get(cvs, 'zoomLevel');
				const width = cvs.scrollWidth;
				const fracX = relX / width;
				const fracY = relY / width;
				const zoomFac = Math.pow(2.0, -0.2 * zoom);
				const moveX = zoomFac * fracX;
				const moveY = zoomFac * fracY;
				let posX = storage.get(cvs, 'posX');
				let posY = storage.get(cvs, 'posY');
				posX -= moveX;
				posY += moveY;
				storage.put(cvs, 'posX', posX);
				storage.put(cvs, 'posY', posY);
				self.refresh();
			}

		}

	};

	/*
	 * This is called when the user moves the mouse over the map.
	 */
	this.mouseMove = function(e) {
		const cvs = e.target;
		const btn = storage.get(cvs, 'mouseButton');

		/*
		 * Move map if mouse button is pressed.
		 */
		if (btn === true) {
			const x = e.offsetX;
			const y = e.offsetY;
			const startX = storage.get(cvs, 'mouseStartX');
			const startY = storage.get(cvs, 'mouseStartY');
			const relX = x - startX;
			const relY = y - startY;
			ui.moveMap(relX, relY);
		}

	};

	/*
	 * This is called when the user moves the scroll wheel over the map.
	 */
	this.scroll = function(e) {
		e.preventDefault();
		const delta = e.deltaY;
		const direction = delta > 0 ? 1 : (delta < 0 ? -1 : 0);
		const cvs = document.getElementById('map_canvas');
		let zoom = storage.get(cvs, 'zoomLevel');
		zoom -= direction;

		/*
		 * Zoom level shall not go below zero.
		 */
		if (zoom < 0) {
			zoom = 0;
		}

		/*
		 * Zoom level shall not go above 120.
		 */
		if (zoom > 120) {
			zoom = 120;
		}

		storage.put(cvs, 'zoomLevel', zoom);

		/*
		 * Perform delayed refresh.
		 */
		const refresh = function() {
			self.refresh();
		};

		let timeout = self._timeoutScroll;
		window.clearTimeout(timeout);
		timeout = window.setTimeout(refresh, 250);
		self._timeoutScroll = timeout;
		ui.moveMap(null, null);
	};

	/*
	 * This is called when the window is resized.
	 */
	this.resize = function() {
		let timeout = self._timeoutResize;
		window.clearTimeout(timeout);

		/*
		 * Perform delayed refresh.
		 */
		const resize = function() {
			self.refresh();
		};

		timeout = window.setTimeout(resize, 100);
		self._timeoutResize = timeout;
	};

	/*
	 * Initializes the (right) side bar of the user interface.
	 */
	this.initializeSidebar = function() {
		const sidebar = document.getElementById('right_sidebar');
		const opener = document.getElementById('right_sidebar_opener');
		const dateFormat = unescape('YYYY-MM-DDThh:mm:ss%B1hh:mm');
		const elemFrom = ui.createElement('From');
		const fieldFrom = document.createElement('input');
		fieldFrom.className = 'textfield';
		fieldFrom.setAttribute('type', 'text');
		fieldFrom.setAttribute('placeholder', dateFormat);
		elemFrom.appendChild(fieldFrom);
		sidebar.appendChild(elemFrom);
		const elemTo = ui.createElement('To');
		const fieldTo = document.createElement('input');
		fieldTo.className = 'textfield';
		fieldTo.setAttribute('type', 'text');
		fieldTo.setAttribute('placeholder', dateFormat);
		elemTo.appendChild(fieldTo);
		sidebar.appendChild(elemTo);
		const elemButtons = ui.createElement('');
		const buttonApply = document.createElement('button');
		buttonApply.className = 'button';
		const buttonApplyCaption = document.createTextNode('Apply');
		buttonApply.appendChild(buttonApplyCaption);

		/*
		 * This is called when the user clicks on the 'Apply' button.
		 */
		buttonApply.onclick = function(e) {
			const valueFrom = helper.cleanValue(fieldFrom.value);
			const valueTo = helper.cleanValue(fieldTo.value);
			const cvs = document.getElementById('map_canvas');
			storage.put(cvs, 'minTime', valueFrom);
			storage.put(cvs, 'maxTime', valueTo);
			self.refresh();
		};

		elemButtons.appendChild(buttonApply);
		const buttonHide = document.createElement('button');
		buttonHide.className = 'button next';
		const buttonHideCaption = document.createTextNode('Hide');
		buttonHide.appendChild(buttonHideCaption);

		/*
		 * This is called when the user clicks on the 'Hide' button.
		 */
		buttonHide.onclick = function(e) {
			sidebar.style.display = 'none';
			opener.style.display = 'block';
		};

		elemButtons.appendChild(buttonHide);
		sidebar.appendChild(elemButtons);
		const elemSpacerA = document.createElement('div');
		elemSpacerA.className = 'vspace';
		sidebar.appendChild(elemSpacerA);
		const elemNorthing = ui.createElement('Northing');
		const fieldNorthing = document.createElement('input');
		fieldNorthing.className = 'textfield';
		fieldNorthing.setAttribute('id', 'northing_field');
		fieldNorthing.setAttribute('readonly', 'readonly');
		elemNorthing.appendChild(fieldNorthing);
		sidebar.appendChild(elemNorthing);
		const elemEasting = ui.createElement('Easting');
		const fieldEasting = document.createElement('input');
		fieldEasting.className = 'textfield';
		fieldEasting.setAttribute('id', 'easting_field');
		fieldEasting.setAttribute('readonly', 'readonly');
		elemEasting.appendChild(fieldEasting);
		sidebar.appendChild(elemEasting);
		const elemZoom = ui.createElement('Zoom');
		const fieldZoom = document.createElement('input');
		fieldZoom.className = 'textfield';
		fieldZoom.setAttribute('id', 'zoom_field');
		fieldZoom.setAttribute('readonly', 'readonly');
		elemZoom.appendChild(fieldZoom);
		sidebar.appendChild(elemZoom);
		const elemSpacerB = document.createElement('div');
		elemSpacerB.className = 'vspace';
		sidebar.appendChild(elemSpacerB);
		const elemNorthingKM = ui.createElement('N [km]');
		const fieldNorthingKM = document.createElement('input');
		fieldNorthingKM.className = 'textfield';
		fieldNorthingKM.setAttribute('id', 'northing_field_km');
		fieldNorthingKM.setAttribute('readonly', 'readonly');
		elemNorthingKM.appendChild(fieldNorthingKM);
		sidebar.appendChild(elemNorthingKM);
		const elemEastingKM = ui.createElement('E [km]');
		const fieldEastingKM = document.createElement('input');
		fieldEastingKM.className = 'textfield';
		fieldEastingKM.setAttribute('id', 'easting_field_km');
		fieldEastingKM.setAttribute('readonly', 'readonly');
		elemEastingKM.appendChild(fieldEastingKM);
		sidebar.appendChild(elemEastingKM);
		const elemSpacerC = document.createElement('div');
		elemSpacerC.className = 'vspace';
		sidebar.appendChild(elemSpacerC);
		const elemLongitude = ui.createElement('Longitude');
		const fieldLongitude = document.createElement('input');
		fieldLongitude.className = 'textfield';
		fieldLongitude.setAttribute('id', 'longitude_field');
		fieldLongitude.setAttribute('readonly', 'readonly');
		elemLongitude.appendChild(fieldLongitude);
		sidebar.appendChild(elemLongitude);
		const elemLatitude = ui.createElement('Latitude');
		const fieldLatitude = document.createElement('input');
		fieldLatitude.className = 'textfield';
		fieldLatitude.setAttribute('id', 'latitude_field');
		fieldLatitude.setAttribute('readonly', 'readonly');
		elemLatitude.appendChild(fieldLatitude);
		sidebar.appendChild(elemLatitude);

		/*
		 * This is called when the user clicks on the sidebar opener.
		 */
		opener.onclick = function(e) {
			opener.style.display = 'none';
			sidebar.style.display = 'block';
		};

	};

	/*
	 * This is called when the user interface initializes.
	 */
	this.initialize = function() {
		const body = document.getElementsByTagName('body')[0];
		const div = document.getElementById('map_div');
		const cvs = document.createElement('canvas');
		cvs.id = 'map_canvas';
		cvs.className = 'mapcanvas';
		storage.put(cvs, 'posX', 0.0);
		storage.put(cvs, 'posY', 0.0);
		storage.put(cvs, 'zoomLevel', 0);
		storage.put(cvs, 'minTime', null);
		storage.put(cvs, 'maxTime', null);
		storage.put(cvs, 'useOSMTiles', true);
		storage.put(cvs, 'imageRequestId', 0);
		storage.put(cvs, 'imageResponseId', 0);
		storage.put(cvs, 'imageZoom', 0);
		storage.put(cvs, 'mouseStartX', 0);
		storage.put(cvs, 'mouseStartY', 0);
		storage.put(cvs, 'touchStartX', 0);
		storage.put(cvs, 'touchStartY', 0);
		storage.put(cvs, 'touchStartDistance', 0);
		storage.put(cvs, 'touchLastX', 0);
		storage.put(cvs, 'touchLastY', 0);
		storage.put(cvs, 'offsetX', 0);
		storage.put(cvs, 'offsetY', 0);
		storage.put(cvs, 'dragInterrupted', false);
		cvs.addEventListener('mousedown', self.mouseDown);
		cvs.addEventListener('mouseup', self.mouseUp);
		cvs.addEventListener('mousemove', self.mouseMove);
		cvs.addEventListener('wheel', self.scroll);
		cvs.addEventListener('touchstart', self.touchStart);
		cvs.addEventListener('touchmove', self.touchMove);
		cvs.addEventListener('touchend', self.touchEnd);
		cvs.addEventListener('touchcancel', self.touchCancel);
		window.addEventListener('resize', self.resize);
		div.appendChild(cvs);
		self.initializeSidebar();
		helper.blockSite(false);
		self.refresh();
	};

}

/*
 * The (global) event handlers.
 */
const handler = new Handler();
document.addEventListener('DOMContentLoaded', handler.initialize);

