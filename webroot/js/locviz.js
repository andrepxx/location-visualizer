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
var globals = new Globals();

/*
 * A class implementing data storage.
 */
function Storage() {
	var g_map = new WeakMap();
	
	/*
	 * Store a value under a key inside an element.
	 */
	this.put = function(elem, key, value) {
		var map = g_map.get(elem);
		
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
		var map = g_map.get(elem);
		
		/*
		 * Check if element is unknown.
		 */
		if (map == null) {
			return null;
		} else {
			var value = map.get(key);
			return value;
		}
		
	};
	
	/*
	 * Check if a certain key exists inside an element.
	 */
	this.has = function(elem, key) {
		var map = g_map.get(elem);
		
		/*
		 * Check if element is unknown.
		 */
		if (map == null) {
			return false;
		} else {
			var value = map.has(key);
			return value;
		}
		
	};
	
	/*
	 * Remove a certain key from an element.
	 */
	this.remove = function(elem, key) {
		var map = g_map.get(elem);
		
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
var storage = new Storage();

/*
 * A class supposed to make life a little easier.
 */
function Helper() {
	
	/*
	 * Blocks or unblocks the site for user interactions.
	 */
	this.blockSite = function(blocked) {
		var blocker = document.getElementById('blocker');
		var displayStyle = '';
		
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
		if (blocker !== null) {
			blocker.style.display = displayStyle;
		}
		
	};
	
	/*
	 * Removes all child nodes from an element.
	 */
	this.clearElement = function(elem) {
		
		/*
		 * As long as the element has child nodes, remove one.
		 */
		while (elem.hasChildNodes()) {
			var child = elem.firstChild;
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
			var obj = JSON.parse(jsonString);
			return obj;
		} catch (ex) {
			return null;
		}
		
	};
	
}

/*
 * The (global) helper object.
 */
var helper = new Helper();

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
		var xhr = new XMLHttpRequest();
		
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
					var content = xhr.responseText;
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
		var img = new Image();
		
		/*
		 * Event handler for load event.
		 */
		var eventSuccess = function() {
			
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
		var eventError = function() {
			
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
		var uri = url + '?' + data;
		img.src = uri;
	};
	
}

/*
 * The (global) Ajax engine.
 */
var ajax = new Ajax();

/*
 * A class implementing a key-value-pair.
 */
function KeyValuePair(key, value) {
	var g_key = key;
	var g_value = value;
	
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
	var g_keyValues = Array();
	
	/*
	 * Append a key-value-pair to a request.
	 */
	this.append = function(key, value) {
		var kv = new KeyValuePair(key, value);
		g_keyValues.push(kv);
	};
	
	/*
	 * Returns the URL encoded data for this request.
	 */
	this.getData = function() {
		var numPairs = g_keyValues.length;
		var s = '';
		
		/*
		 * Iterate over the key-value pairs.
		 */
		for (var i = 0; i < numPairs; i++) {
			var keyValue = g_keyValues[i];
			var key = keyValue.getKey();
			var keyEncoded = encodeURIComponent(key);
			var value = keyValue.getValue();
			var valueEncoded = encodeURIComponent(value);
			
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
	var self = this;
	
	/*
	 * Calculate the IDs and positions of the tiles required to
	 * display a certain portion of the map and their positions
	 * inside the coordinate system.
	 */
	this.calculateTiles = function(xres, yres, zoom, xpos, ypos) {
		var zoomExp = 0.2 * zoom;
		var zoomFac = Math.pow(2.0, zoomExp);
		var zoomFacInv = 1.0 / zoomFac;
		var halfWidth = 0.5 * zoomFacInv;
		var aspectRatio = yres / xres;
		var halfHeight = aspectRatio * halfWidth;
		var minX = xpos - halfWidth;
		var maxX = xpos + halfWidth;
		var minY = ypos - halfHeight;
		var maxY = ypos + halfHeight;
		var tileSize = globals.tileSize;
		var osmZoomFloat = Math.log2(zoomFac * (xres / tileSize))
		var osmZoom = Math.floor(osmZoomFloat);
		
		/*
		 * Limit OSM zoom.
		 */
		if (osmZoom < 0.0) {
			osmZoom = 0.0;
		} else if (osmZoom > 19.0) {
			osmZoom = 19.0;
		}
		
		var maxTileId = (1 << osmZoom) - 1;
		var dxPerTile = Math.pow(2.0, -osmZoom);
		var idxMinX = Math.floor((minX + 0.5) / dxPerTile);
		var idxMaxX = Math.floor((maxX + 0.5) / dxPerTile);
		var idxMinY = Math.floor((0.5 - maxY) / dxPerTile);
		var idxMaxY = Math.floor((0.5 - minY) / dxPerTile);
		var tileDescriptors = [];
		
		/*
		 * Iterate over the Y axis.
		 */
		for (var idxY = idxMinY; idxY <= idxMaxY; idxY++) {
			
			/*
			 * Iterate over the X axis.
			 */
			for (var idxX = idxMinX; idxX <= idxMaxX; idxX++) {
				
				/*
				 * Check if tile ID is valid.
				 */
				if ((idxX >= 0) & (idxX <= maxTileId) & (idxY >= 0) & (idxY <= maxTileId)) {
					var tileMinX = (idxX * dxPerTile) - 0.5;
					var tileMaxX = tileMinX + dxPerTile;
					var tileMaxY = 0.5 - (idxY * dxPerTile);
					var tileMinY = tileMaxY - dxPerTile;
					
					/*
					 * Calculate tile IDs and limits.
					 *
					 * OSM coordinates have X axis to the right and
					 * Y axis downwards.
					 *
					 * Our coordinates have X axis to the right and
					 * Y axis upwards and interval [-0.5, 0.5].
					 */
					var tileDescriptor = {
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
		var x = tileDescriptor.osmX;
		var y = tileDescriptor.osmY;
		var z = tileDescriptor.osmZoom;
		var rq = new Request();
		rq.append('cgi', 'get-tile');
		var xString = x.toString();
		rq.append('x', xString);
		var yString = y.toString();
		rq.append('y', yString);
		var zString = z.toString();
		rq.append('z', zString);
		var cgi = globals.cgi;
		var data = rq.getData();
		
		/*
		 * This is called when the server sends data.
		 */
		var callback = function(img, idResponse) {
			listener(tileDescriptor, img);
		};
		
		ajax.requestImage(cgi, data, callback, false, null);
	};
	
	/*
	 * Draws a tile on the map canvas.
	 */
	this.drawTile = function(tileDescriptor) {
		var img = tileDescriptor.imgData;
		
		/*
		 * Check if current tile has image data attached.
		 */
		if (img !== null) {
			var cvs = document.getElementById('map_canvas');
			var xres = cvs.scrollWidth;
			var yres = cvs.scrollHeight;
			var posX = storage.get(cvs, 'posX');
			var posY = storage.get(cvs, 'posY');
			var zoom = storage.get(cvs, 'zoomLevel');
			var zoomExp = 0.2 * zoom;
			var zoomFac = Math.pow(2.0, zoomExp);
			var zoomFacInv = 1.0 / zoomFac;
			var halfWidth = 0.5 * zoomFacInv;
			var aspectRatio = yres / xres;
			var halfHeight = aspectRatio * halfWidth;
			var minX = posX - halfWidth;
			var maxX = posX + halfWidth;
			var minY = posY - halfHeight;
			var maxY = posY + halfHeight;
			var tileMinX = tileDescriptor.minX;
			var tileMinY = tileDescriptor.minY;
			var tileMaxX = tileDescriptor.maxX;
			var tileMaxY = tileDescriptor.maxY;
			var destX = xres * ((tileMinX - minX) * zoomFac);
			var destY = xres * ((maxY - tileMaxY) * zoomFac);
			var destWidth = xres * ((tileMaxX - tileMinX) * zoomFac);
			var destHeight = xres * ((tileMaxY - tileMinY) * zoomFac);
			var ctx = cvs.getContext('2d');
			ctx.drawImage(img, destX, destY, destWidth, destHeight);
		}
		
	};
	
	/*
	 * This is called when a new map tile has been fetched.
	 */
	this.updateTiles = function(tileDescriptors) {
		var cvs = document.getElementById('map_canvas');
		var width = cvs.scrollWidth;
		var height = cvs.scrollHeight;
		var ctx = cvs.getContext('2d');
		ctx.clearRect(0, 0, width, height);
		var tiles = storage.get(cvs, 'osmTiles');
		
		/*
		 * Check if map tiles have to be drawn.
		 */
		if (tiles !== null) {
			var numTiles = tiles.length;
			
			/*
			 * Draw every single map tile.
			 */
			for (var i = 0; i < numTiles; i++) {
				var currentTile = tiles[i];
				
				/*
				 * Draw tile if map tile is available.
				 */
				if (currentTile !== null) {
					this.drawTile(currentTile);
				}
				
			}
			
		}
		
		var img = storage.get(cvs, 'lastImage');
		
		/*
		 * Check if image overlay has to be drawn.
		 */
		if (img !== null) {
			ctx.drawImage(img, 0, 0);
		}
		
	};
	
	/*
	 * Fetch a list of map tiles and invoke callback on each change.
	 *
	 * - Find first tile ID without image data attached.
	 * - Invoke fetchTile(...) to fetch that tile.
	 *
	 * - When tile was fetched:
	 *   - Associate image data with tile ID.
	 *   - Notify callback that new tile was fetched.
	 *   - Invoke yourself to fetch next missing tile.
	 *
	 * - Process terminates, when all tiles have image data
	 *   attached.
	 */
	this.fetchTiles = function(tileIds, callback) {
		var firstTileToFetch = null;
		
		/*
		 * Iterate over all tiles and find the first that is not yet
		 * fetched.
		 */
		for (var i = 0; i < tileIds.length; i++) {
			var currentTile = tileIds[i];
			var fetched = currentTile.fetched;
			
			/*
			 * Check if this is the first tile to fetch.
			 */
			if ((firstTileToFetch === null) & (fetched === false)) {
				firstTileToFetch = currentTile;
			}
			
		}
		
		/*
		 * Check if we have to fetch a tile.
		 */
		if (firstTileToFetch !== null) {
			
			/*
			 * Internal callback invoked by fetchTile(...).
			 */
			var internalCallback = function(tileDescriptor, img) {
				tileDescriptor.imgData = img;
				tileDescriptor.fetched = true;
				callback(tileIds);
				self.fetchTiles(tileIds, callback);
			};
			
			this.fetchTile(firstTileToFetch, internalCallback);
		}
		
	};
	
	/*
	 * Redraw the map with the same image, but a different offset.
	 */
	this.moveMap = function(xoffs, yoffs) {
		var cvs = document.getElementById('map_canvas');
		var img = storage.get(cvs, 'lastImage');
		
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
			var width = cvs.scrollWidth;
			var height = cvs.scrollHeight;
			var zoomLevelRequested = storage.get(cvs, 'zoomLevel');
			var zoomLevelImage = storage.get(cvs, 'imageZoom');
			var zoomLevelDiff = zoomLevelRequested - zoomLevelImage;
			var zoomFac = Math.pow(2.0, 0.2 * zoomLevelDiff);
			var scaledWidth = width * zoomFac;
			var scaledHeight = height * zoomFac;
			var ctx = cvs.getContext('2d');
			ctx.clearRect(0, 0, width, height);
			var dx = xoffs + (0.5 * (width - scaledWidth));
			var dy = yoffs + (0.5 * (height - scaledHeight));
			ctx.drawImage(img, 0, 0, width, height, dx, dy, scaledWidth, scaledHeight);
		}
		
	};
	
	/*
	 * Updates the image element with a new view of the map.
	 */
	this.updateMap = function(xres, yres, xpos, ypos, zoom, useOSMTiles) {
		var rq = new Request();
		rq.append('cgi', 'render');
		var xresString = xres.toString();
		rq.append('xres', xresString);
		var yresString = yres.toString();
		rq.append('yres', yresString);
		var xposString = xpos.toString();
		rq.append('xpos', xposString);
		var yposString = ypos.toString();
		rq.append('ypos', yposString);
		var zoomString = zoom.toString();
		rq.append('zoom', zoomString);
		rq.append('usebg', 'false');
		var cgi = globals.cgi;
		var data = rq.getData();
		var cvs = document.getElementById('map_canvas');
		var idRequest = storage.get(cvs, 'imageRequestId');
		var currentRequestId = idRequest + 1;
		storage.put(cvs, 'imageRequestId', currentRequestId);
		storage.put(cvs, 'osmTiles', []);
		
		/*
		 * This is called when the server sends data.
		 */
		var callback = function(img, idResponse) {
			var cvs = document.getElementById('map_canvas');
			var lastResponse = storage.get(cvs, 'imageResponseId');
			
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
				var width = cvs.scrollWidth;
				var height = cvs.scrollHeight;
				cvs.width = width;
				cvs.height = height;
				var ctx = cvs.getContext('2d');
				ctx.clearRect(0, 0, width, height);
				ctx.drawImage(img, 0, 0);
				
				/*
				 * Check if we should use OSM tiles.
				 */
				if (useOSMTiles) {
					var tileIds = self.calculateTiles(xres, yres, zoom, xpos, ypos);
					storage.put(cvs, 'osmTiles', tileIds);
					
					/*
					 * Internal callback necessary to have
					 * "this" reference.
					 */
					var internalCallback = function() {
						self.updateTiles();
					};
					
					self.fetchTiles(tileIds, internalCallback);
				}
				
			}
			
		};
		
		ajax.requestImage(cgi, data, callback, true, currentRequestId);
	};
	
}

var ui = new UI();

/*
 * This class implements all handler functions for user interaction.
 */
function Handler() {
	var handler = this;
	this._timeoutScroll = null;
	this._timeoutResize = null;
	
	/*
	 * This is called when the map needs to be refreshed.
	 */
	this.refresh = function() {
		var cvs = document.getElementById('map_canvas');
		var width = cvs.scrollWidth;
		var height = cvs.scrollHeight;
		var posX = storage.get(cvs, 'posX');
		var posY = storage.get(cvs, 'posY');
		var zoom = storage.get(cvs, 'zoomLevel');
		var useOSMTiles = storage.get(cvs, 'useOSMTiles');
		ui.updateMap(width, height, posX, posY, zoom, useOSMTiles);
	};
	
	/*
	 * Extracts coordinates from a single-point touch event.
	 */
	this.touchEventToCoordinates = function(e) {
		var cvs = e.target;
		var rect = cvs.getBoundingClientRect();
		var offsetX = rect.left;
		var offsetY = rect.top;
		var touches = e.targetTouches;
		var numTouches = touches.length;
		var touch = null;
		
		/*
		 * If there are touches, extract the first one.
		 */
		if (numTouches > 0) {
			touch = touches.item(0);
		}
		
		var x = 0.0;
		var y = 0.0;
		
		/*
		 * If a touch was extracted, calculate coordinates relative to
		 * the element position.
		 */
		if (touch !== null) {
			var touchX = touch.pageX;
			x = touchX - offsetX;
			var touchY = touch.pageY;
			y = touchY - offsetY;
		}
		
		var result = {
			x: x,
			y: y
		};
		
		return result;
	};
	
	/*
	 * Extracts the distance from a multi-point touch event.
	 */
	this.touchEventToDistance = function(e) {
		var touches = e.targetTouches;
		var numTouches = touches.length;
		
		/*
		 * If there are two touches, extract them and calculate their distance.
		 */
		if (numTouches === 2) {
			var touchA = touches.item(0);
			var touchB = touches.item(1);
			var xA = touchA.pageX;
			var yA = touchA.pageY;
			var xB = touchB.pageX;
			var yB = touchB.pageY;
			var dX = xB - xA;
			var dY = yB - yA;
			var dXSquared = dX * dX;
			var dYSquared = dY * dY;
			var dSquared = dXSquared + dYSquared;
			var distance = Math.sqrt(dSquared);
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
		var cvs = e.target;
		var touches = e.targetTouches;
		var numTouches = touches.length;
		var singleTouch = (numTouches === 1);
		var multiTouch = (numTouches > 1);
		
		/*
		 * Single touch moves the map, multi touch zooms.
		 */
		if (singleTouch) {
			var coords = handler.touchEventToCoordinates(e);
			var x = coords.x;
			var y = coords.y;
			storage.put(cvs, 'mouseButton', true);
			storage.put(cvs, 'dragInterrupted', false);
			storage.put(cvs, 'touchStartX', x);
			storage.put(cvs, 'touchStartY', y);
			storage.put(cvs, 'touchStartDistance', 0.0);
			storage.put(cvs, 'touchLastX', x);
			storage.put(cvs, 'touchLastY', y);
		} else if (multiTouch) {
			var dist = handler.touchEventToDistance(e);
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
		var cvs = e.target;
		var btn = storage.get(cvs, 'mouseButton');
		
		/*
		 * If mouse button is depressed, move map, otherwise zoom map.
		 */
		if (btn) {
			var interrupted = storage.get(cvs, 'dragInterrupted');
			
			/*
			 * Only care if drag was not interrupted.
			 */
			if (!interrupted) {
				var touches = e.targetTouches;
				var numTouches = touches.length;
				var singleTouch = (numTouches === 1);
				
				/*
				 * Only process single touches, not multi-touch
				 * gestures.
				 */
				if (singleTouch) {
					e.preventDefault();
					var coords = handler.touchEventToCoordinates(e);
					var x = coords.x;
					var y = coords.y;
					storage.put(cvs, 'touchLastX', x);
					storage.put(cvs, 'touchLastY', y);
					var startX = storage.get(cvs, 'touchStartX');
					var startY = storage.get(cvs, 'touchStartY');
					var relX = x - startX;
					var relY = y - startY;
					ui.moveMap(relX, relY);
				}
			
			} else {
				ui.moveMap(0, 0);
			}
			
		} else {
			var touches = e.targetTouches;
			var numTouches = touches.length;
			var multiTouch = (numTouches > 1);
			
			/*
			 * Only process multi-touch gestures, not single
			 * touches.
			 */
			if (multiTouch) {
				e.preventDefault();
				var dist = handler.touchEventToDistance(e);
				var startDist = storage.get(cvs, 'touchStartDistance');
				var frac = dist / startDist;
				var diffZoom = Math.round(5.0 * (Math.log(frac) / Math.log(2.0)));
				var zoom = storage.get(cvs, 'imageZoom');
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
		var cvs = e.target;
		var interrupted = storage.get(cvs, 'dragInterrupted');
		
		/*
		 * Only move map position if drag was not interrupted.
		 */
		if (interrupted) {
			handler.refresh();
		} else {
			var touches = e.targetTouches;
			var numTouches = touches.length;
			var noMoreTouches = (numTouches === 0);
			
			/*
			 * Only commit value after the last finger has
			 * been lifted off.
			 */
			if (noMoreTouches) {
				e.preventDefault();
				var x = storage.get(cvs, 'touchLastX');
				var y = storage.get(cvs, 'touchLastY');
				var startX = storage.get(cvs, 'touchStartX');
				var startY = storage.get(cvs, 'touchStartY');
				var relX = x - startX;
				var relY = y - startY;
				var zoom = storage.get(cvs, 'zoomLevel');
				var width = cvs.scrollWidth;
				var fracX = relX / width;
				var fracY = relY / width;
				var zoomFac = Math.pow(2.0, -0.2 * zoom);
				var moveX = zoomFac * fracX;
				var moveY = zoomFac * fracY;
				var posX = storage.get(cvs, 'posX');
				var posY = storage.get(cvs, 'posY');
				posX -= moveX;
				posY += moveY;
				storage.put(cvs, 'posX', posX);
				storage.put(cvs, 'posY', posY);
				handler.refresh();
			}
			
		}
		
	};
	
	/*
	 * This is called when a user cancels a touch action.
	 */
	this.touchCancel = function(e) {
		var cvs = e.target;
		var btn = storage.get(cvs, 'mouseButton');
		
		/*
		 * Abort action if mouse button was depressed.
		 */
		if (btn) {
			storage.put(cvs, 'dragInterrupted', true);
			ui.moveMap(0, 0);
		}
		
		storage.put(cvs, 'mouseButton', false);
	};
	
	/*
	 * This is called when the user presses the mouse button.
	 */
	this.mouseDown = function(e) {
		var btn = e.button;
		
		/*
		 * Check if left mouse button was depressed.
		 */
		if (btn === 0) {
			var cvs = e.target;
			var x = e.offsetX;
			var y = e.offsetY;
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
		var btn = e.button;
		
		/*
		 * Check if left mouse button was released.
		 */
		if (btn === 0) {
			var cvs = e.target;
			var x = e.offsetX;
			var y = e.offsetY;
			storage.put(cvs, 'mouseButton', false);
			var interrupted = storage.get(cvs, 'dragInterrupted');
			
			/*
			 * Load a new position if drag was not interrupted.
			 */
			if (interrupted === true) {
				ui.moveMap(0, 0);
			} else {
				var startX = storage.get(cvs, 'mouseStartX');
				var startY = storage.get(cvs, 'mouseStartY');
				var relX = x - startX;
				var relY = y - startY;
				var zoom = storage.get(cvs, 'zoomLevel');
				var width = cvs.scrollWidth;
				var fracX = relX / width;
				var fracY = relY / width;
				var zoomFac = Math.pow(2.0, -0.2 * zoom);
				var moveX = zoomFac * fracX;
				var moveY = zoomFac * fracY;
				var posX = storage.get(cvs, 'posX');
				var posY = storage.get(cvs, 'posY');
				posX -= moveX;
				posY += moveY;
				storage.put(cvs, 'posX', posX);
				storage.put(cvs, 'posY', posY);
				handler.refresh();
			}
			
		}
		
	};
	
	/*
	 * This is called when the user moves the mouse over the map.
	 */
	this.mouseMove = function(e) {
		var cvs = e.target;
		var btn = storage.get(cvs, 'mouseButton');
		
		/*
		 * Move map if mouse button is pressed.
		 */
		if (btn === true) {
			var x = e.offsetX;
			var y = e.offsetY;
			var startX = storage.get(cvs, 'mouseStartX');
			var startY = storage.get(cvs, 'mouseStartY');
			var relX = x - startX;
			var relY = y - startY;
			ui.moveMap(relX, relY);
		}
		
	};
	
	/*
	 * This is called when the user moves the scroll wheel over the map.
	 */
	this.scroll = function(e) {
		e.preventDefault();
		var delta = e.deltaY;
		var direction = delta > 0 ? 1 : (delta < 0 ? -1 : 0);
		var cvs = document.getElementById('map_canvas');
		var zoom = storage.get(cvs, 'zoomLevel');
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
		var refresh = function() {
			handler.refresh();
		};
		
		var timeout = handler._timeoutScroll;
		window.clearTimeout(timeout);
		timeout = window.setTimeout(refresh, 250);
		handler._timeoutScroll = timeout;
		ui.moveMap(null, null);
	}
	
	/*
	 * This is called when the window is resized.
	 */
	this.resize = function() {
		var timeout = handler._timeoutResize;
		window.clearTimeout(timeout);
		
		/*
		 * Perform delayed refresh.
		 */
		var resize = function() {
			handler.refresh();
		};
		
		timeout = window.setTimeout(resize, 100);
		handler._timeoutResize = timeout;
	}
	
	/*
	 * This is called when the user interface initializes.
	 */
	this.initialize = function() {
		var body = document.getElementsByTagName('body')[0];
		var div = document.getElementById('map_div');
		var width = div.scrollWidth;
		var height = div.scrollHeight;
		var cvs = document.createElement('canvas');
		cvs.id = 'map_canvas';
		cvs.className = 'mapcanvas';
		storage.put(cvs, 'posX', 0.0);
		storage.put(cvs, 'posY', 0.0);
		storage.put(cvs, 'zoomLevel', 0);
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
		cvs.addEventListener('mousedown', handler.mouseDown);
		cvs.addEventListener('mouseup', handler.mouseUp);
		cvs.addEventListener('mousemove', handler.mouseMove);
		cvs.addEventListener('wheel', handler.scroll);
		cvs.addEventListener('touchstart', handler.touchStart);
		cvs.addEventListener('touchmove', handler.touchMove);
		cvs.addEventListener('touchend', handler.touchEnd);
		cvs.addEventListener('touchcancel', handler.touchCancel);
		window.addEventListener('resize', handler.resize);
		div.appendChild(cvs);
		helper.blockSite(false);
		handler.refresh();
	};
	
}

/*
 * The (global) event handlers.
 */
var handler = new Handler();
document.addEventListener('DOMContentLoaded', handler.initialize);

