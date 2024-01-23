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
	 * This is used to prevent the default action from occuring.
	 */
	this.absorbEvent = function(e) {
		e.stopPropagation();
		e.preventDefault();
		return false;
	};

	/*
	 * Creates a generic UI element container with a label.
	 */
	this.createElement = function(labelCaption, labelWidth) {
		const labelDiv = document.createElement('div');
		labelDiv.className = 'label';

		/*
		 * Check if width is specified.
		 */
		if (labelWidth !== null) {
			labelDiv.style.width = labelWidth;
		}

		const labelNode = document.createTextNode(labelCaption);
		labelDiv.appendChild(labelNode)
		const uiElement = document.createElement('div');
		uiElement.className = 'uielement';
		uiElement.appendChild(labelDiv);
		return uiElement;
	};

	/*
	 * Insert an activity header into the body of a table.
	 */
	this.insertActivityHeader = function(body) {
		const headRowA = document.createElement('tr');
		const headElemEmptyA = document.createElement('td');
		headElemEmptyA.className = 'activitiestablecell tablehead';
		const headElemEmptyAColSpan = document.createAttribute('colspan');
		headElemEmptyAColSpan.value = '2';
		headElemEmptyA.setAttributeNode(headElemEmptyAColSpan);
		headRowA.appendChild(headElemEmptyA);
		const headElemRunning = document.createElement('td');
		headElemRunning.className = 'activitiestablecell tablehead';
		const headElemRunningColSpan = document.createAttribute('colspan');
		headElemRunningColSpan.value = '4';
		headElemRunning.setAttributeNode(headElemRunningColSpan);
		const labelRunning = document.createTextNode('RUNNING');
		headElemRunning.appendChild(labelRunning);
		headRowA.appendChild(headElemRunning);
		const headElemCycling = document.createElement('td');
		headElemCycling.className = 'activitiestablecell tablehead';
		const headElemCyclingColSpan = document.createAttribute('colspan');
		headElemCyclingColSpan.value = '3';
		headElemCycling.setAttributeNode(headElemCyclingColSpan);
		const labelCycling = document.createTextNode('CYCLING');
		headElemCycling.appendChild(labelCycling);
		headRowA.appendChild(headElemCycling);
		const headElemOther = document.createElement('td');
		headElemOther.className = 'activitiestablecell tablehead';
		const labelOther = document.createTextNode('OTHER');
		headElemOther.appendChild(labelOther);
		headRowA.appendChild(headElemOther);
		body.appendChild(headRowA);
		const headElemEmptyB = document.createElement('td');
		headElemEmptyB.className = 'activitiestablecell tablehead';
		const headElemEmptyBColSpan = document.createAttribute('colspan');
		headElemEmptyBColSpan.value = '2';
		headElemEmptyB.setAttributeNode(headElemEmptyBColSpan);
		headRowA.appendChild(headElemEmptyB);
		body.appendChild(headRowA);
		const headRowB = document.createElement('tr');
		const headElemBegin = document.createElement('td');
		headElemBegin.className = 'activitiestablecell tablehead';
		headElemBegin.style.minWidth = '210px';
		const labelBegin = document.createTextNode('Begin');
		headElemBegin.appendChild(labelBegin);
		headRowB.appendChild(headElemBegin);
		const headElemWeightKG = document.createElement('td');
		headElemWeightKG.className = 'activitiestablecell tablehead';
		const labelWeightKG = document.createTextNode('Weight [kg]');
		headElemWeightKG.appendChild(labelWeightKG);
		headRowB.appendChild(headElemWeightKG);
		const headElemRunningDuration = document.createElement('td');
		headElemRunningDuration.className = 'activitiestablecell tablehead';
		headElemRunningDuration.style.minWidth = '80px';
		const labelRunningDuration = document.createTextNode('Duration');
		headElemRunningDuration.appendChild(labelRunningDuration);
		headRowB.appendChild(headElemRunningDuration);
		const headElemRunningDistanceKM = document.createElement('td');
		headElemRunningDistanceKM.className = 'activitiestablecell tablehead';
		const labelRunningDistanceKM = document.createTextNode('Distance [km]');
		headElemRunningDistanceKM.appendChild(labelRunningDistanceKM);
		headRowB.appendChild(headElemRunningDistanceKM);
		const headElemRunningStepCount = document.createElement('td');
		headElemRunningStepCount.className = 'activitiestablecell tablehead';
		const labelRunningStepCount = document.createTextNode('Step count');
		headElemRunningStepCount.appendChild(labelRunningStepCount);
		headRowB.appendChild(headElemRunningStepCount);
		const headElemRunningEnergyKJ = document.createElement('td');
		headElemRunningEnergyKJ.className = 'activitiestablecell tablehead';
		const labelRunningEnergyKJ = document.createTextNode('Energy [kJ]');
		headElemRunningEnergyKJ.appendChild(labelRunningEnergyKJ);
		headRowB.appendChild(headElemRunningEnergyKJ);
		const headElemCyclingDuration = document.createElement('td');
		headElemCyclingDuration.className = 'activitiestablecell tablehead';
		headElemCyclingDuration.style.minWidth = '80px';
		const labelCyclingDuration = document.createTextNode('Duration');
		headElemCyclingDuration.appendChild(labelCyclingDuration);
		headRowB.appendChild(headElemCyclingDuration);
		const headElemCyclingDistanceKM = document.createElement('td');
		headElemCyclingDistanceKM.className = 'activitiestablecell tablehead';
		const labelCyclingDistanceKM = document.createTextNode('Distance [km]');
		headElemCyclingDistanceKM.appendChild(labelCyclingDistanceKM);
		headRowB.appendChild(headElemCyclingDistanceKM);
		const headElemCyclingEnergyKJ = document.createElement('td');
		headElemCyclingEnergyKJ.className = 'activitiestablecell tablehead';
		const labelCyclingEnergyKJ = document.createTextNode('Energy [kJ]');
		headElemCyclingEnergyKJ.appendChild(labelCyclingEnergyKJ);
		headRowB.appendChild(headElemCyclingEnergyKJ);
		const headElemOtherEnergyKJ = document.createElement('td');
		headElemOtherEnergyKJ.className = 'activitiestablecell tablehead';
		const labelOtherEnergyKJ = document.createTextNode('Energy [kJ]');
		headElemOtherEnergyKJ.appendChild(labelOtherEnergyKJ);
		headRowB.appendChild(headElemOtherEnergyKJ);
		const headElemEdit = document.createElement('td');
		headElemEdit.className = 'activitiestablecell tablehead';
		const labelEdit = document.createTextNode('Edit');
		headElemEdit.appendChild(labelEdit);
		headRowB.appendChild(headElemEdit);
		const headElemRemove = document.createElement('td');
		headElemRemove.className = 'activitiestablecell tablehead';
		headElemRemove.style.minWidth = '140px';
		const labelRemove = document.createTextNode('Remove');
		headElemRemove.appendChild(labelRemove);
		headRowB.appendChild(headElemRemove);
		body.appendChild(headRowB);
	};

	/*
	 * Function to parse activities and display them as a table
	 * inside a div element.
	 */
	this.displayActivities = function(div, response) {
		helper.clearElement(div);
		const table = document.createElement('table');
		table.className = 'activitiestable';
		const body = document.createElement('tbody');
		const activities = response.Activities;
		const numActivities = activities.length;
		let allRemoveLinks = [];
		let allYesNoLinks = [];
		let currentYearAndMonth = '';

		/*
		 * Iterate over all activities.
		 */
		for (let i = 0; i < numActivities; i++) {
			const activity = activities[i];
			const row = document.createElement('tr');
			row.className = 'activitiestablerow';
			const begin = activity.Begin;
			const beginLength = begin.length;

			/*
			 * Check if beginning time stamp contains at least year and month.
			 */
			if (beginLength >= 7) {
				const yearAndMonth = begin.substring(0, 7);

				/*
				 * If year and month changed, insert another header.
				 */
				if (yearAndMonth != currentYearAndMonth) {
					this.insertActivityHeader(body);
					currentYearAndMonth = yearAndMonth;
				}

			}

			const beginElem = document.createElement('td');
			beginElem.className = 'activitiestablecell';
			const beginDiv = document.createElement('div');
			beginDiv.className = 'link';
			beginDiv.style.display = 'inline-block';
			const beginNode = document.createTextNode(begin);
			beginDiv.appendChild(beginNode);

			/*
			 * This is called when the user clicks on the begin time.
			 */
			beginDiv.onclick = function(e) {
				const cvs = document.getElementById('map_canvas');
				const valueFrom = activity.Begin;
				const valueTo = activity.End;
				const fieldFrom = document.getElementById('right_sidebar_field_from');
				fieldFrom.value = valueFrom;
				const fieldTo = document.getElementById('right_sidebar_field_to');
				fieldTo.value = valueTo;
				storage.put(cvs, 'minTime', valueFrom);
				storage.put(cvs, 'maxTime', valueTo);
				handler.refresh();
			};

			beginElem.appendChild(beginDiv);
			row.appendChild(beginElem);
			const weightKG = activity.WeightKG;
			const weightKGElem = document.createElement('td');
			weightKGElem.className = 'activitiestablecell';
			const weightKGNode = document.createTextNode(weightKG);
			weightKGElem.appendChild(weightKGNode);
			row.appendChild(weightKGElem);
			const running = activity.Running;
			const runningZero = running.Zero;

			/*
			 * Insert blank or running information.
			 */
			if (runningZero === true) {
				const runningElem = document.createElement('td');
				runningElem.className = 'activitiestablecell';
				const runningColSpan = document.createAttribute('colspan');
				runningColSpan.value = '4';
				runningElem.setAttributeNode(runningColSpan);
				row.appendChild(runningElem);
			} else {
				const runningDuration = running.Duration;
				const runningDurationElem = document.createElement('td');
				runningDurationElem.className = 'activitiestablecell';
				const runningDurationNode = document.createTextNode(runningDuration);
				runningDurationElem.appendChild(runningDurationNode);
				row.appendChild(runningDurationElem);
				const runningDistanceKM = running.DistanceKM;
				const runningDistanceKMElem = document.createElement('td');
				runningDistanceKMElem.className = 'activitiestablecell';
				const runningDistanceKMNode = document.createTextNode(runningDistanceKM);
				runningDistanceKMElem.appendChild(runningDistanceKMNode);
				row.appendChild(runningDistanceKMElem);
				const runningStepCount = running.StepCount;
				const runningStepCountString = runningStepCount.toString();
				const runningStepCountElem = document.createElement('td');
				runningStepCountElem.className = 'activitiestablecell';
				const runningStepCountNode = document.createTextNode(runningStepCountString);
				runningStepCountElem.appendChild(runningStepCountNode);
				row.appendChild(runningStepCountElem);
				const runningEnergyKJ = running.EnergyKJ;
				const runningEnergyKJString = runningEnergyKJ.toString();
				const runningEnergyKJElem = document.createElement('td');
				runningEnergyKJElem.className = 'activitiestablecell';
				const runningEnergyKJNode = document.createTextNode(runningEnergyKJString);
				runningEnergyKJElem.appendChild(runningEnergyKJNode);
				row.appendChild(runningEnergyKJElem);
			}

			const cycling = activity.Cycling;
			const cyclingZero = cycling.Zero;

			/*
			 * Insert blank or cycling information.
			 */
			if (cyclingZero === true) {
				const cyclingElem = document.createElement('td');
				cyclingElem.className = 'activitiestablecell';
				const cyclingColSpan = document.createAttribute('colspan');
				cyclingColSpan.value = '3';
				cyclingElem.setAttributeNode(cyclingColSpan);
				row.appendChild(cyclingElem);
			} else {
				const cyclingDuration = cycling.Duration;
				const cyclingDurationElem = document.createElement('td');
				cyclingDurationElem.className = 'activitiestablecell';
				const cyclingDurationNode = document.createTextNode(cyclingDuration);
				cyclingDurationElem.appendChild(cyclingDurationNode);
				row.appendChild(cyclingDurationElem);
				const cyclingDistanceKM = cycling.DistanceKM;
				const cyclingDistanceKMElem = document.createElement('td');
				cyclingDistanceKMElem.className = 'activitiestablecell';
				const cyclingDistanceKMNode = document.createTextNode(cyclingDistanceKM);
				cyclingDistanceKMElem.appendChild(cyclingDistanceKMNode);
				row.appendChild(cyclingDistanceKMElem);
				const cyclingEnergyKJ = cycling.EnergyKJ;
				const cyclingEnergyKJString = cyclingEnergyKJ.toString();
				const cyclingEnergyKJElem = document.createElement('td');
				cyclingEnergyKJElem.className = 'activitiestablecell';
				const cyclingEnergyKJNode = document.createTextNode(cyclingEnergyKJString);
				cyclingEnergyKJElem.appendChild(cyclingEnergyKJNode);
				row.appendChild(cyclingEnergyKJElem);
			}

			const other = activity.Other;
			const otherZero = other.Zero;

			/*
			 * Insert blank or other information.
			 */
			if (otherZero === true) {
				const otherElem = document.createElement('td');
				otherElem.className = 'activitiestablecell';
				row.appendChild(otherElem);
			} else {
				const otherEnergyKJ = other.EnergyKJ;
				const otherEnergyKJString = otherEnergyKJ.toString();
				const otherEnergyKJElem = document.createElement('td');
				otherEnergyKJElem.className = 'activitiestablecell';
				const otherEnergyKJNode = document.createTextNode(otherEnergyKJString);
				otherEnergyKJElem.appendChild(otherEnergyKJNode);
				row.appendChild(otherEnergyKJElem);
			}

			const editElem = document.createElement('td');
			editElem.className = 'activitiestablecell';
			editElem.style.textAlign = 'left';
			const editLink = document.createElement('div');
			editLink.className = 'link';
			const editCaption = document.createTextNode('Edit');
			editLink.appendChild(editCaption);

			/*
			 * Open dialog for editing when user clicks 'edit'.
			 */
			editLink.onclick = function(e) {
				self.displayActivitiesEditDialog(response, i);
			};

			editElem.appendChild(editLink);
			row.appendChild(editElem);
			const removeElem = document.createElement('td');
			removeElem.className = 'activitiestablecell';
			removeElem.style.textAlign = 'left';
			const removeQuestionDiv = document.createElement('div');
			allYesNoLinks.push(removeQuestionDiv);
			removeQuestionDiv.style.display = 'none';
			const removeQuestionCaption = document.createTextNode('Remove?');
			removeQuestionDiv.appendChild(removeQuestionCaption);
			removeElem.appendChild(removeQuestionDiv);
			const removeLinkYes = document.createElement('div');
			allYesNoLinks.push(removeLinkYes);
			removeLinkYes.className = 'link';
			removeLinkYes.style.display = 'none';
			removeLinkYes.style.paddingLeft = '5px';
			const removeCaptionYes = document.createTextNode('yes');
			removeLinkYes.appendChild(removeCaptionYes);

			/*
			 * Remove entry when user clicks 'yes'.
			 */
			removeLinkYes.onclick = function(e) {
				const cgi = globals.cgi;
				const request = new Request();
				request.append('cgi', 'remove-activity');
				const id = i.toString();
				request.append('id', id);
				const revision = response.Revision;
				const revisionString = revision.toString();
				request.append('revision', revisionString);
				const cvs = document.getElementById('map_canvas');
				const token = storage.get(cvs, 'token');
				request.append('token', token);
				const data = request.getData();
				const mime = globals.mimeDefault;

				/*
				 * This is called when the server returns a response.
				 */
				const callback = function(content) {
					handler.showActivities();
				};

				ajax.request('POST', cgi, data, mime, callback, false);
			};

			removeElem.appendChild(removeLinkYes);
			const removeLinkNo = document.createElement('div');
			allYesNoLinks.push(removeLinkNo);
			removeLinkNo.className = 'link';
			removeLinkNo.style.display = 'none';
			removeLinkNo.style.paddingLeft = '5px';
			const removeCaptionNo = document.createTextNode('no');
			removeLinkNo.appendChild(removeCaptionNo);

			/*
			 * Hide both options when user clicks 'no'.
			 */
			removeLinkNo.onclick = function(e) {
				removeQuestionDiv.style.display = 'none';
				removeLinkYes.style.display = 'none';
				removeLinkNo.style.display = 'none';
				removeLink.style.display = 'inline-block';
			};

			removeElem.appendChild(removeLinkNo);
			const removeLink = document.createElement('div');
			allRemoveLinks.push(removeLink);
			removeLink.className = 'link';
			removeLink.style.display = 'inline-block';
			const removeCaption = document.createTextNode('Remove');
			removeLink.appendChild(removeCaption);

			/*
			 * This is called when the user clicks on the 'remove' link.
			 */
			removeLink.onclick = function(e) {

				/*
				 * Show all remove links.
				 */
				for (let i = 0; i < allRemoveLinks.length; i++) {
					const link = allRemoveLinks[i];
					link.style.display = 'inline-block';
				}

				/*
				 * Hide all yes / no links.
				 */
				for (let i = 0; i < allYesNoLinks.length; i++) {
					const link = allYesNoLinks[i];
					link.style.display = 'none';
				}

				removeQuestionDiv.style.display = 'inline-block';
				removeLinkYes.style.display = 'inline-block';
				removeLinkNo.style.display = 'inline-block';
				removeLink.style.display = 'none';
			};

			removeElem.appendChild(removeLink);
			row.appendChild(removeElem);
			body.appendChild(row);
		}

		this.insertActivityHeader(body);
		table.appendChild(body);
		div.appendChild(table);
		const spacerDivA = document.createElement('div');
		spacerDivA.className = 'vspace';
		div.appendChild(spacerDivA);

		/*
		 * Insert label and value into statistics table.
		 */
		const insertIntoTable = function(body, label, value) {
			const tr = document.createElement('tr');
			tr.className = 'activitiestablerow';
			const labelTd = document.createElement('td');
			labelTd.className = 'activitiestablecell labelcell';
			const labelNode = document.createTextNode(label);
			labelTd.appendChild(labelNode);
			tr.appendChild(labelTd);
			const valueTd = document.createElement('td');
			valueTd.className = 'activitiestablecell';
			const valueNode = document.createTextNode(value);
			valueTd.appendChild(valueNode);
			tr.appendChild(valueTd);
			body.appendChild(tr);
		};

		const statistics = response.Statistics;
		const statisticsTable = document.createElement('table');
		statisticsTable.className = 'activitiestable';
		const statisticsBody = document.createElement('tbody');
		const running = statistics.Running;
		const runningDuration = running.Duration;
		const runningDurationString = runningDuration.toString();
		insertIntoTable(statisticsBody, 'Running duration', runningDurationString);
		const runningDistanceKM = running.DistanceKM;
		const runningDistanceKMString = runningDistanceKM.toString();
		insertIntoTable(statisticsBody, 'Running distance [km]', runningDistanceKMString);
		const runningStepCount = running.StepCount;
		const runningStepCountString = runningStepCount.toString();
		insertIntoTable(statisticsBody, 'Running step count', runningStepCountString);
		const runningEnergyKJ = running.EnergyKJ;
		const runningEnergyKJString = runningEnergyKJ.toString();
		insertIntoTable(statisticsBody, 'Running energy [kJ]', runningEnergyKJString);
		const cycling = statistics.Cycling;
		const cyclingDuration = cycling.Duration;
		const cyclingDurationString = cyclingDuration.toString();
		insertIntoTable(statisticsBody, 'Cycling duration', cyclingDurationString);
		const cyclingDistanceKM = cycling.DistanceKM;
		const cyclingDistanceKMString = cyclingDistanceKM.toString();
		insertIntoTable(statisticsBody, 'Cycling distance [km]', cyclingDistanceKMString);
		const cyclingEnergyKJ = cycling.EnergyKJ;
		const cyclingEnergyKJString = cyclingEnergyKJ.toString();
		insertIntoTable(statisticsBody, 'Cycling energy [kJ]', cyclingEnergyKJString);
		const other = statistics.Other;
		const otherEnergyKJ = other.EnergyKJ;
		const otherEnergyKJString = otherEnergyKJ.toString();
		insertIntoTable(statisticsBody, 'Other energy [kJ]', otherEnergyKJString);
		statisticsTable.appendChild(statisticsBody);
		div.appendChild(statisticsTable);
		const spacerDivB = document.createElement('div');
		spacerDivB.className = 'vspace';
		div.appendChild(spacerDivB);
		const buttonDiv = document.createElement('div');
		const buttonAdd = document.createElement('button');
		buttonAdd.className = 'button';
		const buttonAddCaption = document.createTextNode('Add');
		buttonAdd.appendChild(buttonAddCaption);

		/*
		 * This is called when the user clicks on the 'Add' button.
		 */
		buttonAdd.onclick = function(e) {
			self.displayActivitiesAddDialog();
		};

		buttonDiv.appendChild(buttonAdd);
		const buttonImport = document.createElement('button');
		buttonImport.className = 'button next';
		const buttonImportCaption = document.createTextNode('Import');
		buttonImport.appendChild(buttonImportCaption);

		/*
		 * This is called when the user clicks on the 'Import' button.
		 */
		buttonImport.onclick = function(e) {
			self.displayActivitiesImportDialog();
		};

		buttonDiv.appendChild(buttonImport);
		const buttonBack = document.createElement('button');
		buttonBack.className = 'button next';
		const buttonBackCaption = document.createTextNode('Back');
		buttonBack.appendChild(buttonBackCaption);

		/*
		 * This is called when the user clicks on the 'Back' button.
		 */
		buttonBack.onclick = function(e) {
			div.style.display = 'none';
		};

		buttonDiv.appendChild(buttonBack);
		div.appendChild(buttonDiv);
		div.style.display = 'block';
		div.focus();
	};

	/*
	 * Display dialog to add an activity.
	 */
	this.displayActivitiesAddDialog = function() {
		const div = document.getElementById('activitiesdialog');
		const innerDiv = document.getElementById('activitiesdialog_content');
		const dateFormat = unescape('YYYY-MM-DDThh:mm:ss%B1hh:mm');
		const elemBegin = this.createElement('Begin', '180px');
		const fieldBegin = document.createElement('input');
		fieldBegin.className = 'textfield';
		fieldBegin.setAttribute('type', 'text');
		fieldBegin.setAttribute('placeholder', dateFormat);
		elemBegin.appendChild(fieldBegin);
		innerDiv.appendChild(elemBegin);
		const elemWeightKG = this.createElement('Weight [kg]', '180px');
		const fieldWeightKG = document.createElement('input');
		fieldWeightKG.className = 'textfield rightalign';
		fieldWeightKG.setAttribute('type', 'text');
		fieldWeightKG.setAttribute('placeholder', '75.0');
		elemWeightKG.appendChild(fieldWeightKG);
		innerDiv.appendChild(elemWeightKG);
		const elemRunningDuration = this.createElement('Running duration', '180px');
		const fieldRunningDuration = document.createElement('input');
		fieldRunningDuration.className = 'textfield rightalign';
		fieldRunningDuration.setAttribute('type', 'text');
		fieldRunningDuration.setAttribute('placeholder', '1h30m');
		elemRunningDuration.appendChild(fieldRunningDuration);
		innerDiv.appendChild(elemRunningDuration);
		const elemRunningDistanceKM = this.createElement('Running distance [km]', '180px');
		const fieldRunningDistanceKM = document.createElement('input');
		fieldRunningDistanceKM.className = 'textfield rightalign';
		fieldRunningDistanceKM.setAttribute('type', 'text');
		fieldRunningDistanceKM.setAttribute('placeholder', '15.0');
		elemRunningDistanceKM.appendChild(fieldRunningDistanceKM);
		innerDiv.appendChild(elemRunningDistanceKM);
		const elemRunningStepCount = this.createElement('Running step count', '180px');
		const fieldRunningStepCount = document.createElement('input');
		fieldRunningStepCount.className = 'textfield rightalign';
		fieldRunningStepCount.setAttribute('type', 'text');
		fieldRunningStepCount.setAttribute('placeholder', '18000');
		elemRunningStepCount.appendChild(fieldRunningStepCount);
		innerDiv.appendChild(elemRunningStepCount);
		const elemRunningEnergyKJ = this.createElement('Running energy [kJ]', '180px');
		const fieldRunningEnergyKJ = document.createElement('input');
		fieldRunningEnergyKJ.className = 'textfield rightalign';
		fieldRunningEnergyKJ.setAttribute('type', 'text');
		fieldRunningEnergyKJ.setAttribute('placeholder', '10000');
		elemRunningEnergyKJ.appendChild(fieldRunningEnergyKJ);
		innerDiv.appendChild(elemRunningEnergyKJ);
		const elemCyclingDuration = this.createElement('Cycling duration', '180px');
		const fieldCyclingDuration = document.createElement('input');
		fieldCyclingDuration.className = 'textfield rightalign';
		fieldCyclingDuration.setAttribute('type', 'text');
		fieldCyclingDuration.setAttribute('placeholder', '1h30m');
		elemCyclingDuration.appendChild(fieldCyclingDuration);
		innerDiv.appendChild(elemCyclingDuration);
		const elemCyclingDistanceKM = this.createElement('Cycling distance [km]', '180px');
		const fieldCyclingDistanceKM = document.createElement('input');
		fieldCyclingDistanceKM.className = 'textfield rightalign';
		fieldCyclingDistanceKM.setAttribute('type', 'text');
		fieldCyclingDistanceKM.setAttribute('placeholder', '45.0');
		elemCyclingDistanceKM.appendChild(fieldCyclingDistanceKM);
		innerDiv.appendChild(elemCyclingDistanceKM);
		const elemCyclingEnergyKJ = this.createElement('Cycling energy [kJ]', '180px');
		const fieldCyclingEnergyKJ = document.createElement('input');
		fieldCyclingEnergyKJ.className = 'textfield rightalign';
		fieldCyclingEnergyKJ.setAttribute('type', 'text');
		fieldCyclingEnergyKJ.setAttribute('placeholder', '10000');
		elemCyclingEnergyKJ.appendChild(fieldCyclingEnergyKJ);
		innerDiv.appendChild(elemCyclingEnergyKJ);
		const elemOtherEnergyKJ = this.createElement('Other energy [kJ]', '180px');
		const fieldOtherEnergyKJ = document.createElement('input');
		fieldOtherEnergyKJ.className = 'textfield rightalign';
		fieldOtherEnergyKJ.setAttribute('type', 'text');
		fieldOtherEnergyKJ.setAttribute('placeholder', '5000');
		elemOtherEnergyKJ.appendChild(fieldOtherEnergyKJ);
		innerDiv.appendChild(elemOtherEnergyKJ);
		const buttonAdd = document.createElement('button');
		buttonAdd.className = 'button';
		const buttonAddCaption = document.createTextNode('Add');
		buttonAdd.appendChild(buttonAddCaption);

		/*
		 * This is called when the user clicks on the 'Add' button.
		 */
		buttonAdd.onclick = function(e) {
			const cgi = globals.cgi;
			const request = new Request();
			request.append('cgi', 'add-activity');
			const begin = fieldBegin.value;
			request.append('begin', begin);
			const weightKG = fieldWeightKG.value;
			request.append('weightkg', weightKG);
			const runningDuration = fieldRunningDuration.value;
			request.append('runningduration', runningDuration);
			const runningDistanceKM = fieldRunningDistanceKM.value;
			request.append('runningdistancekm', runningDistanceKM);
			const runningStepCount = fieldRunningStepCount.value;
			request.append('runningstepcount', runningStepCount);
			const runningEnergyKJ = fieldRunningEnergyKJ.value;
			request.append('runningenergykj', runningEnergyKJ);
			const cyclingDuration = fieldCyclingDuration.value;
			request.append('cyclingduration', cyclingDuration);
			const cyclingDistanceKM = fieldCyclingDistanceKM.value;
			request.append('cyclingdistancekm', cyclingDistanceKM);
			const cyclingEnergyKJ = fieldCyclingEnergyKJ.value;
			request.append('cyclingenergykj', cyclingEnergyKJ);
			const otherEnergyKJ = fieldOtherEnergyKJ.value;
			request.append('otherenergykj', otherEnergyKJ);
			const cvs = document.getElementById('map_canvas');
			const token = storage.get(cvs, 'token');
			request.append('token', token);
			const data = request.getData();
			const mime = globals.mimeDefault;

			/*
			 * This is called when the server returns a response.
			 */
			const callback = function(content) {
				const response = helper.parseJSON(content);
				const success = response.Success;

				/*
				 * Check if logout was successful.
				 */
				if (success === true) {
					div.style.display = 'none';
					helper.clearElement(innerDiv);
					handler.showActivities();
				}

			};

			ajax.request('POST', cgi, data, mime, callback, false);
		};

		innerDiv.appendChild(buttonAdd);
		const buttonCancel = document.createElement('button');
		buttonCancel.className = 'button';
		const buttonCancelCaption = document.createTextNode('Cancel');
		buttonCancel.appendChild(buttonCancelCaption);

		/*
		 * This is called when the user clicks on the 'Cancel' button.
		 */
		buttonCancel.onclick = function(e) {
			div.style.display = 'none';
			helper.clearElement(innerDiv);
		};

		innerDiv.appendChild(buttonCancel);
		div.style.display = 'block';
		fieldBegin.focus();
	};

	/*
	 * Display dialog to edit an activity.
	 */
	this.displayActivitiesEditDialog = function(response, idx) {
		const activities = response.Activities;
		const activity = activities[idx];
		const div = document.getElementById('activitiesdialog');
		const innerDiv = document.getElementById('activitiesdialog_content');
		const dateFormat = unescape('YYYY-MM-DDThh:mm:ss%B1hh:mm');
		const valueBegin = activity.Begin;
		const valueBeginString = valueBegin.toString();
		const elemBegin = this.createElement('Begin', '180px');
		const fieldBegin = document.createElement('input');
		fieldBegin.className = 'textfield';
		fieldBegin.setAttribute('type', 'text');
		fieldBegin.value = valueBeginString;
		elemBegin.appendChild(fieldBegin);
		innerDiv.appendChild(elemBegin);
		const valueWeightKG = activity.WeightKG;
		const valueWeightKGString = valueWeightKG.toString();
		const elemWeightKG = this.createElement('Weight [kg]', '180px');
		const fieldWeightKG = document.createElement('input');
		fieldWeightKG.className = 'textfield rightalign';
		fieldWeightKG.setAttribute('type', 'text');
		fieldWeightKG.value = valueWeightKGString;
		elemWeightKG.appendChild(fieldWeightKG);
		innerDiv.appendChild(elemWeightKG);
		const runningActivity = activity.Running;
		const valueRunningDuration = runningActivity.Duration;
		const valueRunningDurationString = valueRunningDuration.toString();
		const elemRunningDuration = this.createElement('Running duration', '180px');
		const fieldRunningDuration = document.createElement('input');
		fieldRunningDuration.className = 'textfield rightalign';
		fieldRunningDuration.setAttribute('type', 'text');
		fieldRunningDuration.value = valueRunningDurationString;
		elemRunningDuration.appendChild(fieldRunningDuration);
		innerDiv.appendChild(elemRunningDuration);
		const valueRunningDistanceKM = runningActivity.DistanceKM;
		const valueRunningDistanceKMString = valueRunningDistanceKM.toString();
		const elemRunningDistanceKM = this.createElement('Running distance [km]', '180px');
		const fieldRunningDistanceKM = document.createElement('input');
		fieldRunningDistanceKM.className = 'textfield rightalign';
		fieldRunningDistanceKM.setAttribute('type', 'text');
		fieldRunningDistanceKM.value = valueRunningDistanceKMString;
		elemRunningDistanceKM.appendChild(fieldRunningDistanceKM);
		innerDiv.appendChild(elemRunningDistanceKM);
		const valueRunningStepCount = runningActivity.StepCount;
		const valueRunningStepCountString = valueRunningStepCount.toString();
		const elemRunningStepCount = this.createElement('Running step count', '180px');
		const fieldRunningStepCount = document.createElement('input');
		fieldRunningStepCount.className = 'textfield rightalign';
		fieldRunningStepCount.setAttribute('type', 'text');
		fieldRunningStepCount.value = valueRunningStepCountString;
		elemRunningStepCount.appendChild(fieldRunningStepCount);
		innerDiv.appendChild(elemRunningStepCount);
		const valueRunningEnergyKJ = runningActivity.EnergyKJ;
		const valueRunningEnergyKJString = valueRunningEnergyKJ.toString();
		const elemRunningEnergyKJ = this.createElement('Running energy [kJ]', '180px');
		const fieldRunningEnergyKJ = document.createElement('input');
		fieldRunningEnergyKJ.className = 'textfield rightalign';
		fieldRunningEnergyKJ.setAttribute('type', 'text');
		fieldRunningEnergyKJ.value = valueRunningEnergyKJString;
		elemRunningEnergyKJ.appendChild(fieldRunningEnergyKJ);
		innerDiv.appendChild(elemRunningEnergyKJ);
		const cyclingActivity = activity.Cycling;
		const valueCyclingDuration = cyclingActivity.Duration;
		const valueCyclingDurationString = valueCyclingDuration.toString();
		const elemCyclingDuration = this.createElement('Cycling duration', '180px');
		const fieldCyclingDuration = document.createElement('input');
		fieldCyclingDuration.className = 'textfield rightalign';
		fieldCyclingDuration.setAttribute('type', 'text');
		fieldCyclingDuration.value = valueCyclingDurationString;
		elemCyclingDuration.appendChild(fieldCyclingDuration);
		innerDiv.appendChild(elemCyclingDuration);
		const valueCyclingDistanceKM = cyclingActivity.DistanceKM;
		const valueCyclingDistanceKMString = valueCyclingDistanceKM.toString();
		const elemCyclingDistanceKM = this.createElement('Cycling distance [km]', '180px');
		const fieldCyclingDistanceKM = document.createElement('input');
		fieldCyclingDistanceKM.className = 'textfield rightalign';
		fieldCyclingDistanceKM.setAttribute('type', 'text');
		fieldCyclingDistanceKM.value = valueCyclingDistanceKMString;
		elemCyclingDistanceKM.appendChild(fieldCyclingDistanceKM);
		innerDiv.appendChild(elemCyclingDistanceKM);
		const valueCyclingEnergyKJ = cyclingActivity.EnergyKJ;
		const valueCyclingEnergyKJString = valueCyclingEnergyKJ.toString();
		const elemCyclingEnergyKJ = this.createElement('Cycling energy [kJ]', '180px');
		const fieldCyclingEnergyKJ = document.createElement('input');
		fieldCyclingEnergyKJ.className = 'textfield rightalign';
		fieldCyclingEnergyKJ.setAttribute('type', 'text');
		fieldCyclingEnergyKJ.value = valueCyclingEnergyKJString;
		elemCyclingEnergyKJ.appendChild(fieldCyclingEnergyKJ);
		innerDiv.appendChild(elemCyclingEnergyKJ);
		const otherActivity = activity.Other;
		const valueOtherEnergyKJ = otherActivity.EnergyKJ;
		const valueOtherEnergyKJString = valueOtherEnergyKJ.toString();
		const elemOtherEnergyKJ = this.createElement('Other energy [kJ]', '180px');
		const fieldOtherEnergyKJ = document.createElement('input');
		fieldOtherEnergyKJ.className = 'textfield rightalign';
		fieldOtherEnergyKJ.setAttribute('type', 'text');
		fieldOtherEnergyKJ.value = valueOtherEnergyKJString;
		elemOtherEnergyKJ.appendChild(fieldOtherEnergyKJ);
		innerDiv.appendChild(elemOtherEnergyKJ);
		const buttonEdit = document.createElement('button');
		buttonEdit.className = 'button';
		const buttonEditCaption = document.createTextNode('Edit');
		buttonEdit.appendChild(buttonEditCaption);

		/*
		 * This is called when the user clicks on the 'Edit' button.
		 */
		buttonEdit.onclick = function(e) {
			const cgi = globals.cgi;
			const request = new Request();
			request.append('cgi', 'replace-activity');
			const id = idx.toString();
			request.append('id', id);
			const revision = response.Revision;
			request.append('revision', revision);
			const begin = fieldBegin.value;
			request.append('begin', begin);
			const weightKG = fieldWeightKG.value;
			request.append('weightkg', weightKG);
			const runningDuration = fieldRunningDuration.value;
			request.append('runningduration', runningDuration);
			const runningDistanceKM = fieldRunningDistanceKM.value;
			request.append('runningdistancekm', runningDistanceKM);
			const runningStepCount = fieldRunningStepCount.value;
			request.append('runningstepcount', runningStepCount);
			const runningEnergyKJ = fieldRunningEnergyKJ.value;
			request.append('runningenergykj', runningEnergyKJ);
			const cyclingDuration = fieldCyclingDuration.value;
			request.append('cyclingduration', cyclingDuration);
			const cyclingDistanceKM = fieldCyclingDistanceKM.value;
			request.append('cyclingdistancekm', cyclingDistanceKM);
			const cyclingEnergyKJ = fieldCyclingEnergyKJ.value;
			request.append('cyclingenergykj', cyclingEnergyKJ);
			const otherEnergyKJ = fieldOtherEnergyKJ.value;
			request.append('otherenergykj', otherEnergyKJ);
			const cvs = document.getElementById('map_canvas');
			const token = storage.get(cvs, 'token');
			request.append('token', token);
			const data = request.getData();
			const mime = globals.mimeDefault;

			/*
			 * This is called when the server returns a response.
			 */
			const callback = function(content) {
				const response = helper.parseJSON(content);
				const success = response.Success;

				/*
				 * Check if logout was successful.
				 */
				if (success === true) {
					div.style.display = 'none';
					helper.clearElement(innerDiv);
					handler.showActivities();
				}

			};

			ajax.request('POST', cgi, data, mime, callback, false);
		};

		innerDiv.appendChild(buttonEdit);
		const buttonCancel = document.createElement('button');
		buttonCancel.className = 'button';
		const buttonCancelCaption = document.createTextNode('Cancel');
		buttonCancel.appendChild(buttonCancelCaption);

		/*
		 * This is called when the user clicks on the 'Cancel' button.
		 */
		buttonCancel.onclick = function(e) {
			div.style.display = 'none';
			helper.clearElement(innerDiv);
		};

		innerDiv.appendChild(buttonCancel);
		div.style.display = 'block';
		fieldBegin.focus();
	};

	/*
	 * Display dialog to import activities.
	 */
	this.displayActivitiesImportDialog = function() {
		const div = document.getElementById('activitiesdialog');
		const innerDiv = document.getElementById('activitiesdialog_content');
		const importArea = document.createElement('textarea');
		importArea.className = 'textarea';
		innerDiv.appendChild(importArea);
		const buttonsDiv = document.createElement('div');
		const buttonImport = document.createElement('button');
		buttonImport.className = 'button';
		const buttonImportCaption = document.createTextNode('Import');
		buttonImport.appendChild(buttonImportCaption);

		/*
		 * This is called when the user clicks on the 'Import' button.
		 */
		buttonImport.onclick = function(e) {
			const cgi = globals.cgi;
			const request = new Request();
			request.append('cgi', 'import-activity-csv');
			const importData = importArea.value;
			request.append('data', importData);
			const cvs = document.getElementById('map_canvas');
			const token = storage.get(cvs, 'token');
			request.append('token', token);
			const data = request.getData();
			const mime = globals.mimeDefault;

			/*
			 * This is called when the server returns a response.
			 */
			const callback = function(content) {
				const response = helper.parseJSON(content);
				const success = response.Success;

				/*
				 * Check if logout was successful.
				 */
				if (success === true) {
					div.style.display = 'none';
					helper.clearElement(innerDiv);
					handler.showActivities();
				}

			};

			ajax.request('POST', cgi, data, mime, callback, false);
		};

		buttonsDiv.appendChild(buttonImport);
		const buttonCancel = document.createElement('button');
		buttonCancel.className = 'button';
		const buttonCancelCaption = document.createTextNode('Cancel');
		buttonCancel.appendChild(buttonCancelCaption);

		/*
		 * This is called when the user clicks on the 'Cancel' button.
		 */
		buttonCancel.onclick = function(e) {
			div.style.display = 'none';
			helper.clearElement(innerDiv);
		};

		buttonsDiv.appendChild(buttonCancel);
		innerDiv.appendChild(buttonsDiv);
		div.style.display = 'block';
		importArea.focus();
	};

	/*
	 * This is called when the user drops a geo data file into the upload area.
	 */
	this.uploadGeoData = function(e) {
		e.stopPropagation();
		e.preventDefault();
		const transfer = e.dataTransfer;
		const files = transfer.files;
		const numFiles = files.length;

		/*
		 * Check if there is a file.
		 */
		if (numFiles > 0) {
			const file = files[0];
			const importFormatField = document.getElementById('geodb_import_format_field');
			const importFormatValue = importFormatField.value;
			const importStrategyField = document.getElementById('geodb_import_strategy_field');
			const importStrategyValue = importStrategyField.value;
			const importSortField = document.getElementById('geodb_sort_field');
			const importSortValue = importSortField.value;

			/*
			 * This gets called when the server returns a response.
			 */
			let responseHandler = function(response) {
				ui.displayGeoDBImportStats(response);
			};

			const url = globals.cgi;
			const data = new FormData();
			data.append('cgi', 'import-geodata');
			data.append('format', importFormatValue);
			data.append('strategy', importStrategyValue);
			data.append('sort', importSortValue);
			const cvs = document.getElementById('map_canvas');
			const token = storage.get(cvs, 'token');
			data.append('token', token);
			data.append('file', file);
			ajax.request('POST', url, data, null, responseHandler, true);
		}

		return false;
	};

	/*
	 * Parse GeoDB import stats and display them.
	 */
	this.displayGeoDBImportStats = function(content) {
		const response = helper.parseJSON(content);
		const contentDiv = document.getElementById('geodbdialog_content');
		helper.clearElement(contentDiv);
		const tableDiv = document.createElement('div');
		const status = response.Status;
		const success = status.Success;

		/*
		 * Only render table if import was successful.
		 */
		if (success === true) {
			const table = document.createElement('table');
			table.className = 'geodbtable';
			const body = document.createElement('tbody');
			const columnNames = ['Before', 'Source', 'Imported', 'After'];
			const rowNames = ['LocationCount', 'Ordered', 'OrderedStrict', 'TimestampEarliest', 'TimestampLatest'];
			const rowLabels = ['Location count', 'Ordered', 'Ordered strict', 'Timestamp earliest', 'Timestamp latest'];
			const headerRow = document.createElement('tr');
			headerRow.className = 'geodbtablerow';
			const emptyHeaderCell = document.createElement('td');
			emptyHeaderCell.className = 'geodbtablecell tablehead';
			headerRow.appendChild(emptyHeaderCell);

			/*
			 * Fill header row.
			 */
			for (let i = 0; i < columnNames.length; i++) {
				const headerCell = document.createElement('td');
				headerCell.className = 'geodbtablecell tablehead';
				const headerCellLabel = columnNames[i];
				const headerCellNode = document.createTextNode(headerCellLabel);
				headerCell.appendChild(headerCellNode);
				headerRow.appendChild(headerCell);
			}

			body.appendChild(headerRow);

			/*
			 * Fill data rows.
			 */
			for (let i = 0; i < rowNames.length; i++) {
				const dataRow = document.createElement('tr');
				dataRow.className = 'geodbtablerow';
				const rowName = rowNames[i];
				const rowLabel = rowLabels[i];
				const rowLabelCell = document.createElement('td');
				rowLabelCell.className = 'geodbtablecell tablehead';
				const rowLabelNode = document.createTextNode(rowLabel);
				rowLabelCell.appendChild(rowLabelNode);
				dataRow.appendChild(rowLabelCell)

				/*
				 * Fill data columns.
				 */
				for (let j = 0; j < columnNames.length; j++) {
					const dataCell = document.createElement('td');
					dataCell.className = 'geodbtablecell rightalign';
					const columnName = columnNames[j];
					const dataCellContent = response[columnName][rowName];
					const dataCellContentString = dataCellContent.toString();
					const dataCellNode = document.createTextNode(dataCellContentString);
					dataCell.appendChild(dataCellNode);
					dataRow.appendChild(dataCell);
				}

				body.appendChild(dataRow);
			}

			table.appendChild(body);
			tableDiv.appendChild(table);
		}

		contentDiv.appendChild(tableDiv);
		const buttonDiv = document.createElement('div');
		const buttonBack = document.createElement('button');
		buttonBack.className = 'button';
		const buttonBackCaption = document.createTextNode('Back');
		buttonBack.appendChild(buttonBackCaption);

		/*
		 * This is called when the user clicks on the 'Back' button.
		 */
		buttonBack.onclick = function(e) {
			div.style.display = 'none';
			handler.showGeoDB();
		};

		buttonDiv.appendChild(buttonBack);
		contentDiv.appendChild(buttonDiv);
		const div = document.getElementById('geodbdialog');
		div.style.display = 'block';
		buttonBack.focus();
	}

	/*
	 * Parse GeoDB stats and display them as a table inside a div element.
	 */
	this.displayGeoDBStats = function(div, response) {
		const cgi = globals.cgi;
		const cvs = document.getElementById('map_canvas');
		const token = storage.get(cvs, 'token');
		const cgiDownloadGeoDBContent = 'download-geodb-content';
		helper.clearElement(div);
		const table = document.createElement('table');
		table.className = 'geodbtable';
		const body = document.createElement('tbody');
		const labels = ['Location count', 'Ordered', 'Ordered strict', 'Timestamp earliest', 'Timestamp latest'];
		const locationCount = response.LocationCount;
		const locationCountString = locationCount.toString();
		const ordered = response.Ordered;
		const orderedString = ordered.toString();
		const orderedStrict = response.OrderedStrict;
		const orderedStrictString = orderedStrict.toString();
		const timestampEarliest = response.TimestampEarliest;
		const timestampEarliestString = timestampEarliest.toString();
		const timestampLatest = response.TimestampLatest;
		const timestampLatestString = timestampLatest.toString();
		const values = [locationCountString, orderedString, orderedStrictString, timestampEarliestString, timestampLatestString];

		/*
		 * Iterate over all labels and values and add them to table.
		 */
		for (let i = 0; i < values.length; i++) {
			const row = document.createElement('tr');
			row.className = 'geodbtablerow';
			const labelDiv = document.createElement('td');
			labelDiv.className = 'geodbtablecell';
			const label = labels[i];
			const labelNode = document.createTextNode(label);
			labelDiv.appendChild(labelNode);
			row.appendChild(labelDiv);
			const valueDiv = document.createElement('td');
			valueDiv.className = 'geodbtablecell rightalign';
			const value = values[i];
			const valueNode = document.createTextNode(value);
			valueDiv.appendChild(valueNode);
			row.appendChild(valueDiv);
			body.appendChild(row);
		}

		table.appendChild(body);
		div.appendChild(table);
		const spacerDivA = document.createElement('div');
		spacerDivA.className = 'vspace';
		div.appendChild(spacerDivA);
		const downloadLinksDiv = document.createElement('div');
		const downloadLinkBinaryDiv = document.createElement('div');
		const downloadLinkBinary = document.createElement('a');
		downloadLinkBinary.className = 'link';
		const requestDownloadBinary = new Request();
		requestDownloadBinary.append('cgi', cgiDownloadGeoDBContent);
		requestDownloadBinary.append('format', 'binary');
		requestDownloadBinary.append('token', token);
		const requestDownloadBinaryData = requestDownloadBinary.getData();
		const downloadLinkBinaryHref = document.createAttribute('href');
		downloadLinkBinaryHref.value = cgi + '?' + requestDownloadBinaryData;
		downloadLinkBinary.setAttributeNode(downloadLinkBinaryHref);
		const downloadLinkBinaryNode = document.createTextNode('Download binary (*.geodb)');
		downloadLinkBinary.appendChild(downloadLinkBinaryNode);
		downloadLinkBinaryDiv.appendChild(downloadLinkBinary);
		downloadLinksDiv.appendChild(downloadLinkBinaryDiv);
		const downloadLinkCSVDiv = document.createElement('div');
		const downloadLinkCSV = document.createElement('a');
		downloadLinkCSV.className = 'link';
		const requestDownloadCSV = new Request();
		requestDownloadCSV.append('cgi', cgiDownloadGeoDBContent);
		requestDownloadCSV.append('format', 'csv');
		requestDownloadCSV.append('token', token);
		const requestDownloadCSVData = requestDownloadCSV.getData();
		const downloadLinkCSVHref = document.createAttribute('href');
		downloadLinkCSVHref.value = cgi + '?' + requestDownloadCSVData;
		downloadLinkCSV.setAttributeNode(downloadLinkCSVHref);
		const downloadLinkCSVNode = document.createTextNode('Download CSV (*.csv)');
		downloadLinkCSV.appendChild(downloadLinkCSVNode);
		downloadLinkCSVDiv.appendChild(downloadLinkCSV);
		downloadLinksDiv.appendChild(downloadLinkCSVDiv);
		const downloadLinkGPXDiv = document.createElement('div');
		const downloadLinkGPX = document.createElement('a');
		downloadLinkGPX.className = 'link';
		const requestDownloadGPX = new Request();
		requestDownloadGPX.append('cgi', cgiDownloadGeoDBContent);
		requestDownloadGPX.append('format', 'gpx');
		requestDownloadGPX.append('token', token);
		const requestDownloadGPXData = requestDownloadGPX.getData();
		const downloadLinkGPXHref = document.createAttribute('href');
		downloadLinkGPXHref.value = cgi + '?' + requestDownloadGPXData;
		downloadLinkGPX.setAttributeNode(downloadLinkGPXHref);
		const downloadLinkGPXNode = document.createTextNode('Download compact GPX (*.gpx)');
		downloadLinkGPX.appendChild(downloadLinkGPXNode);
		downloadLinkGPXDiv.appendChild(downloadLinkGPX);
		downloadLinksDiv.appendChild(downloadLinkGPXDiv);
		const downloadLinkGPXPrettyDiv = document.createElement('div');
		const downloadLinkGPXPretty = document.createElement('a');
		downloadLinkGPXPretty.className = 'link';
		const requestDownloadGPXPretty = new Request();
		requestDownloadGPXPretty.append('cgi', cgiDownloadGeoDBContent);
		requestDownloadGPXPretty.append('format', 'gpx-pretty');
		requestDownloadGPXPretty.append('token', token);
		const requestDownloadGPXPrettyData = requestDownloadGPXPretty.getData();
		const downloadLinkGPXPrettyHref = document.createAttribute('href');
		downloadLinkGPXPrettyHref.value = cgi + '?' + requestDownloadGPXPrettyData;
		downloadLinkGPXPretty.setAttributeNode(downloadLinkGPXPrettyHref);
		const downloadLinkGPXPrettyNode = document.createTextNode('Download pretty-printed GPX (*.gpx)');
		downloadLinkGPXPretty.appendChild(downloadLinkGPXPrettyNode);
		downloadLinkGPXPrettyDiv.appendChild(downloadLinkGPXPretty);
		downloadLinksDiv.appendChild(downloadLinkGPXPrettyDiv);
		const downloadLinkJSONDiv = document.createElement('div');
		const downloadLinkJSON = document.createElement('a');
		downloadLinkJSON.className = 'link';
		const requestDownloadJSON = new Request();
		requestDownloadJSON.append('cgi', cgiDownloadGeoDBContent);
		requestDownloadJSON.append('format', 'json');
		requestDownloadJSON.append('token', token);
		const requestDownloadJSONData = requestDownloadJSON.getData();
		const downloadLinkJSONHref = document.createAttribute('href');
		downloadLinkJSONHref.value = cgi + '?' + requestDownloadJSONData;
		downloadLinkJSON.setAttributeNode(downloadLinkJSONHref);
		const downloadLinkJSONNode = document.createTextNode('Download compact JSON (*.json)');
		downloadLinkJSON.appendChild(downloadLinkJSONNode);
		downloadLinkJSONDiv.appendChild(downloadLinkJSON);
		downloadLinksDiv.appendChild(downloadLinkJSONDiv);
		const downloadLinkJSONPrettyDiv = document.createElement('div');
		const downloadLinkJSONPretty = document.createElement('a');
		downloadLinkJSONPretty.className = 'link';
		const requestDownloadJSONPretty = new Request();
		requestDownloadJSONPretty.append('cgi', cgiDownloadGeoDBContent);
		requestDownloadJSONPretty.append('format', 'json-pretty');
		requestDownloadJSONPretty.append('token', token);
		const requestDownloadJSONPrettyData = requestDownloadJSONPretty.getData();
		const downloadLinkJSONPrettyHref = document.createAttribute('href');
		downloadLinkJSONPrettyHref.value = cgi + '?' + requestDownloadJSONPrettyData;
		downloadLinkJSONPretty.setAttributeNode(downloadLinkJSONPrettyHref);
		const downloadLinkJSONPrettyNode = document.createTextNode('Download pretty-printed JSON (*.json)');
		downloadLinkJSONPretty.appendChild(downloadLinkJSONPrettyNode);
		downloadLinkJSONPrettyDiv.appendChild(downloadLinkJSONPretty);
		downloadLinksDiv.appendChild(downloadLinkJSONPrettyDiv);
		div.appendChild(downloadLinksDiv);
		const spacerDivB = document.createElement('div');
		spacerDivB.className = 'vspace';
		div.appendChild(spacerDivB);
		const importPropertiesDiv = document.createElement('div');
		const importPropertiesDescriptionDiv = document.createElement('div');
		const importPropertiesDescriptionNode = document.createTextNode('Drop GPX or GeoJSON file to import location data into geographical database.');
		importPropertiesDescriptionDiv.appendChild(importPropertiesDescriptionNode);
		importPropertiesDiv.appendChild(importPropertiesDescriptionDiv);
		const importFormatElem = this.createElement('Format', '180px');
		const importFormatLabels = ['GPS Exchange (*.gpx)', 'GeoJSON (*.json)'];
		const importFormatValues = ['gpx', 'json'];
		const importFormatDefault = importFormatValues[1];
		const fieldImportFormat = document.createElement('select');

		/*
		 * Add supported values for import format.
		 */
		for (let i = 0; i < importFormatValues.length; i++) {
			const l = importFormatLabels[i];
			const v = importFormatValues[i];
			const option = document.createElement('option');
			option.setAttribute('value', v);
			const optionNode = document.createTextNode(l);
			option.appendChild(optionNode);
			fieldImportFormat.appendChild(option);
		}

		fieldImportFormat.className = 'textfield';
		fieldImportFormat.setAttribute('id', 'geodb_import_format_field');
		fieldImportFormat.value = importFormatDefault;
		importFormatElem.appendChild(fieldImportFormat);
		importPropertiesDiv.appendChild(importFormatElem);
		const importStrategyElem = this.createElement('Import strategy', '180px');
		const importStrategyValues = ['all', 'newer', 'none'];
		const importStrategyDefault = importStrategyValues[1];
		const fieldImportStrategy = document.createElement('select');

		/*
		 * Add supported values for import strategy.
		 */
		for (let i = 0; i < importStrategyValues.length; i++) {
			const v = importStrategyValues[i];
			const option = document.createElement('option');
			option.setAttribute('value', v);
			const optionNode = document.createTextNode(v);
			option.appendChild(optionNode);
			fieldImportStrategy.appendChild(option);
		}

		fieldImportStrategy.className = 'textfield';
		fieldImportStrategy.setAttribute('id', 'geodb_import_strategy_field');
		fieldImportStrategy.value = importStrategyDefault;
		importStrategyElem.appendChild(fieldImportStrategy);
		importPropertiesDiv.appendChild(importStrategyElem);
		const sortElem = this.createElement('Sort', '180px');
		const sortValues = ['true', 'false'];
		const sortDefault = sortValues[0];
		const fieldSort = document.createElement('select');

		/*
		 * Add supported values for sorting.
		 */
		for (let i = 0; i < sortValues.length; i++) {
			const v = sortValues[i];
			const option = document.createElement('option');
			option.setAttribute('value', v);
			const optionNode = document.createTextNode(v);
			option.appendChild(optionNode);
			fieldSort.appendChild(option);
		}

		fieldSort.className = 'textfield';
		fieldSort.setAttribute('id', 'geodb_sort_field');
		fieldSort.value = sortDefault;
		sortElem.appendChild(fieldSort);
		importPropertiesDiv.appendChild(sortElem);
		div.appendChild(importPropertiesDiv);
		const spacerDivC = document.createElement('div');
		spacerDivB.className = 'vspace';
		div.appendChild(spacerDivC);
		const buttonDiv = document.createElement('div');
		const buttonBack = document.createElement('button');
		buttonBack.className = 'button';
		const buttonBackCaption = document.createTextNode('Back');
		buttonBack.appendChild(buttonBackCaption);

		/*
		 * This is called when the user clicks on the 'Back' button.
		 */
		buttonBack.onclick = function(e) {
			div.style.display = 'none';
		};

		buttonDiv.appendChild(buttonBack);
		div.appendChild(buttonDiv);
		div.addEventListener('dragend', this.absorbEvent);
		div.addEventListener('dragenter', this.absorbEvent);
		div.addEventListener('dragleave', this.absorbEvent);
		div.addEventListener('dragover', this.absorbEvent);
		div.addEventListener('drop', this.uploadGeoData);
		div.style.display = 'block';
		fieldImportFormat.focus();
	};

	/*
	 * Initializes the (right) side bar of the user interface.
	 */
	this.initializeSidebar = function() {
		const sidebar = document.getElementById('right_sidebar');
		const opener = document.getElementById('right_sidebar_opener');
		const dateFormat = unescape('YYYY-MM-DDThh:mm:ss%B1hh:mm');
		const elemFrom = this.createElement('From', null);
		const fieldFrom = document.createElement('input');
		fieldFrom.className = 'textfield';
		fieldFrom.setAttribute('id', 'right_sidebar_field_from');
		fieldFrom.setAttribute('type', 'text');
		fieldFrom.setAttribute('placeholder', dateFormat);
		elemFrom.appendChild(fieldFrom);
		sidebar.appendChild(elemFrom);
		const elemTo = this.createElement('To', null);
		const fieldTo = document.createElement('input');
		fieldTo.setAttribute('id', 'right_sidebar_field_to');
		fieldTo.className = 'textfield';
		fieldTo.setAttribute('type', 'text');
		fieldTo.setAttribute('placeholder', dateFormat);
		elemTo.appendChild(fieldTo);
		sidebar.appendChild(elemTo);
		const elemMapIntensity = this.createElement('M. intens.', null);
		const fieldMapIntensity = document.createElement('select');

		/*
		 * Add values of supported spread factors.
		 */
		for (let i = 0; i <= 10; i++) {
			const v = i.toString();
			const option = document.createElement('option');
			option.setAttribute('value', v);
			const optionNode = document.createTextNode(v);
			option.appendChild(optionNode);
			fieldMapIntensity.appendChild(option);
		}

		fieldMapIntensity.className = 'textfield';
		fieldMapIntensity.setAttribute('id', 'map_intensity_field');
		fieldMapIntensity.value = '5';
		elemMapIntensity.appendChild(fieldMapIntensity);
		sidebar.appendChild(elemMapIntensity);
		const elemSpread = this.createElement('Spread', null);
		const fieldSpread = document.createElement('select');

		/*
		 * Add values of supported spread factors.
		 */
		for (let i = 0; i < 5; i++) {
			const v = i.toString();
			const option = document.createElement('option');
			option.setAttribute('value', v);
			const optionNode = document.createTextNode(v);
			option.appendChild(optionNode);
			fieldSpread.appendChild(option);
		}

		fieldSpread.className = 'textfield';
		fieldSpread.setAttribute('id', 'spread_field');
		fieldSpread.value = '0';
		elemSpread.appendChild(fieldSpread);
		sidebar.appendChild(elemSpread);
		const elemColorMapping = this.createElement('Color map.', null);
		const fieldColorMapping = document.createElement('select');
		fieldColorMapping.className = 'textfield';
		const colors = ['(default)', 'red', 'green', 'blue', 'yellow', 'cyan', 'magenta', 'gray', 'brightblue', 'white'];

		/*
		 * Iterate over the colors and add them to dropdown.
		 */
		for (let i = 0; i < colors.length; i++) {
			const color = colors[i];
			const option = document.createElement('option');
			option.setAttribute('value', color);
			const optionNode = document.createTextNode(color);
			option.appendChild(optionNode);
			fieldColorMapping.appendChild(option);
		}

		elemColorMapping.appendChild(fieldColorMapping);
		sidebar.appendChild(elemColorMapping);
		const elemButtonsA = this.createElement('', null);
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
			const valueMapIntensity = helper.cleanValue(fieldMapIntensity.value);
			const valueSpread = helper.cleanValue(fieldSpread.value);
			const valueFgColor = helper.cleanValue(fieldColorMapping.value);
			const cvs = document.getElementById('map_canvas');
			storage.put(cvs, 'colorScale', valueMapIntensity);
			storage.put(cvs, 'spread', valueSpread);
			storage.put(cvs, 'fgColor', valueFgColor);
			storage.put(cvs, 'minTime', valueFrom);
			storage.put(cvs, 'maxTime', valueTo);
			handler.refresh();
		};

		elemButtonsA.appendChild(buttonApply);
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

		elemButtonsA.appendChild(buttonHide);
		const buttonActivities = document.createElement('button');
		buttonActivities.className = 'button next';
		const buttonActivitiesCaption = document.createTextNode('Activities');
		buttonActivities.appendChild(buttonActivitiesCaption);

		/*
		 * This is called when the user clicks on the 'Activities' button.
		 */
		buttonActivities.onclick = function(e) {
			handler.showActivities();
		};

		elemButtonsA.appendChild(buttonActivities);
		const buttonLogout = document.createElement('button');
		buttonLogout.className = 'button buttonred nextgap';
		const buttonLogoutCaption = document.createTextNode('Logout');
		buttonLogout.appendChild(buttonLogoutCaption);

		/*
		 * This is called when the user clicks on the 'Logout' button.
		 */
		buttonLogout.onclick = function(e) {
			sidebar.style.display = 'none';
			opener.style.display = 'block';
			handler.logout();
		};

		elemButtonsA.appendChild(buttonLogout);
		sidebar.appendChild(elemButtonsA);
		const elemButtonsB = this.createElement('');
		const buttonGeoDB = document.createElement('button');
		buttonGeoDB.className = 'button';
		const buttonGeoDBCaption = document.createTextNode('GeoDB');
		buttonGeoDB.appendChild(buttonGeoDBCaption);

		/*
		 * This is called when the user clicks on the 'GeoDB' button.
		 */
		buttonGeoDB.onclick = function(e) {
			handler.showGeoDB();
		};

		elemButtonsB.appendChild(buttonGeoDB);
		const buttonFullscreen = document.createElement('button');
		buttonFullscreen.className = 'button next';
		const buttonFullscreenCaption = document.createTextNode('Fullscreen');
		buttonFullscreen.appendChild(buttonFullscreenCaption);

		/*
		 * This is called when the user clicks on the 'Fullscreen' button.
		 */
		buttonFullscreen.onclick = function(e) {

			/*
			 * If we are in fullscreen mode, leave it, otherwise enter it.
			 */
			if (window.matchMedia('(display-mode: fullscreen)').matches) {
			    	document.exitFullscreen();
			} else {
				const elem = document.documentElement;
				elem.requestFullscreen();
	 		}

		};

		elemButtonsB.appendChild(buttonFullscreen);
		sidebar.appendChild(elemButtonsB);
		const elemSpacerA = document.createElement('div');
		elemSpacerA.className = 'vspace';
		sidebar.appendChild(elemSpacerA);
		const elemSpacerB = document.createElement('div');
		elemSpacerB.className = 'vspace';
		sidebar.appendChild(elemSpacerB);
		const elemNorthing = this.createElement('Northing', null);
		const fieldNorthing = document.createElement('input');
		fieldNorthing.className = 'textfield rightalign';
		fieldNorthing.setAttribute('id', 'northing_field');
		fieldNorthing.setAttribute('readonly', 'readonly');
		elemNorthing.appendChild(fieldNorthing);
		sidebar.appendChild(elemNorthing);
		const elemEasting = this.createElement('Easting', null);
		const fieldEasting = document.createElement('input');
		fieldEasting.className = 'textfield rightalign';
		fieldEasting.setAttribute('id', 'easting_field');
		fieldEasting.setAttribute('readonly', 'readonly');
		elemEasting.appendChild(fieldEasting);
		sidebar.appendChild(elemEasting);
		const elemZoom = this.createElement('Zoom', null);
		const fieldZoom = document.createElement('input');
		fieldZoom.className = 'textfield rightalign';
		fieldZoom.setAttribute('id', 'zoom_field');
		fieldZoom.setAttribute('readonly', 'readonly');
		elemZoom.appendChild(fieldZoom);
		sidebar.appendChild(elemZoom);
		const elemSpacerC = document.createElement('div');
		elemSpacerC.className = 'vspace';
		sidebar.appendChild(elemSpacerC);
		const elemNorthingKM = this.createElement('N [km]', null);
		const fieldNorthingKM = document.createElement('input');
		fieldNorthingKM.className = 'textfield rightalign';
		fieldNorthingKM.setAttribute('id', 'northing_field_km');
		fieldNorthingKM.setAttribute('readonly', 'readonly');
		elemNorthingKM.appendChild(fieldNorthingKM);
		sidebar.appendChild(elemNorthingKM);
		const elemEastingKM = this.createElement('E [km]', null);
		const fieldEastingKM = document.createElement('input');
		fieldEastingKM.className = 'textfield rightalign';
		fieldEastingKM.setAttribute('id', 'easting_field_km');
		fieldEastingKM.setAttribute('readonly', 'readonly');
		elemEastingKM.appendChild(fieldEastingKM);
		sidebar.appendChild(elemEastingKM);
		const elemSpacerD = document.createElement('div');
		elemSpacerD.className = 'vspace';
		sidebar.appendChild(elemSpacerD);
		const elemLongitude = this.createElement('Longitude', null);
		const fieldLongitude = document.createElement('input');
		fieldLongitude.className = 'textfield';
		fieldLongitude.setAttribute('id', 'longitude_field');
		fieldLongitude.setAttribute('readonly', 'readonly');
		elemLongitude.appendChild(fieldLongitude);
		sidebar.appendChild(elemLongitude);
		const elemLatitude = this.createElement('Latitude', null);
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
	 * This initializes the login dialog.
	 */
	this.initializeLogin = function(callback) {
		const loginContent = document.getElementById('login_content');
		const elemUser = this.createElement('User', null);
		const fieldUser = document.createElement('input');
		fieldUser.className = 'textfield';
		fieldUser.setAttribute('type', 'text');
		elemUser.appendChild(fieldUser);
		loginContent.appendChild(elemUser);
		const elemPassword = this.createElement('Password', null);
		const fieldPassword = document.createElement('input');
		fieldPassword.className = 'textfield';
		fieldPassword.setAttribute('type', 'password');
		fieldPassword.setAttribute('autocomplete', 'current-password');
		elemPassword.appendChild(fieldPassword);
		loginContent.appendChild(elemPassword);
		const elemButtons = this.createElement('', null);
		const buttonLogin = document.createElement('button');
		buttonLogin.className = 'button';
		const buttonLoginCaption = document.createTextNode('Login');
		buttonLogin.appendChild(buttonLoginCaption);
		elemButtons.appendChild(buttonLogin);
		loginContent.appendChild(elemButtons);

		/*
		 * This is called when the user types in the user name field.
		 */
		fieldUser.onkeyup = function(e) {

			/*
			 * On Enter, advance to password field.
			 */
			if (e.key === 'Enter') {
				fieldPassword.focus();
			}

		}

		/*
		 * This is called when the user types in the password field.
		 */
		fieldPassword.onkeyup = function(e) {

			/*
			 * On Enter, perform login sequence.
			 */
			if (e.key === 'Enter') {
				buttonLogin.click();
			}

		}

		/*
		 * This is called when the user clicks on the 'Login' button.
		 */
		buttonLogin.onclick = function(e) {
			const cgi = globals.cgi;
			const valueUser = helper.cleanValue(fieldUser.value);
			const valuePassword = fieldPassword.value;
			const rqChallenge = new Request();
			rqChallenge.append('cgi', 'auth-request');
			rqChallenge.append('name', valueUser);
			const dataChallenge = rqChallenge.getData();
			const mime = globals.mimeDefault;

			/*
			 * This is called when the server sends a challenge.
			 */
			const callbackChallenge = function(content) {
				const challenge = helper.parseJSON(content);
				const challengeSuccess = challenge.Success;

				/*
				 * Check if challenge could be obtained.
				 */
				if (challengeSuccess === true) {
					const nonceEncoded = challenge.Nonce;
					const nonce = atob(nonceEncoded);

					/*
					 * Function to convert character to integer.
					 */
					const charToInt = function(c) {
						return c.charCodeAt(0);
					};

					const nonceArray = Uint8Array.from(nonce, charToInt);
					const saltEncoded = challenge.Salt;
					const salt = atob(saltEncoded);
					const saltArray = Uint8Array.from(salt, charToInt);
					const encoder = new TextEncoder('utf-8');
					const passwordArray = encoder.encode(valuePassword);
					const shaA = new jsSHA('SHA-512', 'UINT8ARRAY');
					shaA.update(passwordArray);
					const passwordHash = shaA.getHash('UINT8ARRAY');
					const shaB = new jsSHA('SHA-512', 'UINT8ARRAY');
					shaB.update(saltArray);
					shaB.update(passwordHash);
					const innerHash = shaB.getHash('UINT8ARRAY');
					const shaC = new jsSHA('SHA-512', 'UINT8ARRAY');
					shaC.update(nonceArray);
					shaC.update(innerHash);
					const outerHash = shaC.getHash('B64');
					const rqResponse = new Request();
					rqResponse.append('cgi', 'auth-response');
					rqResponse.append('name', valueUser);
					rqResponse.append('hash', outerHash);
					const dataResponse = rqResponse.getData();

					/*
					 * This is called when the server sends a session token.
					 */
					const callbackToken = function(content) {
						const token = helper.parseJSON(content);
						const tokenSuccess = token.Success;

						/*
						 * Check if token could be obtained.
						 */
						if (tokenSuccess === true) {
							fieldPassword.value = '';
							const tokenData = token.Token;
							callback(tokenData);
						}

					}

					ajax.request('POST', cgi, dataResponse, mime, callbackToken, false);
				}

			};

			ajax.request('POST', cgi, dataChallenge, mime, callbackChallenge, false);
		};

	};

	/*
	 * Show the login window.
	 */
	this.showLogin = function() {
		const loginDiv = document.getElementById('login');
		loginDiv.style.display = 'block';
		loginDiv.focus();
	};

	/*
	 * Hide the login window.
	 */
	this.hideLogin = function() {
		const loginDiv = document.getElementById('login');
		loginDiv.style.display = 'none';
	};

	/*
	 * Calculate the IDs and positions of the tiles required to
	 * display a certain portion of the map and their positions
	 * inside the coordinate system.
	 */
	this.calculateTiles = function(xres, yres, zoom, xpos, ypos, colorScale) {
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
						colorScale: colorScale,
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
	this.fetchTile = function(token, tileDescriptor, listener) {
		const x = tileDescriptor.osmX;
		const y = tileDescriptor.osmY;
		const z = tileDescriptor.osmZoom;
		const colorScale = tileDescriptor.colorScale;
		const rq = new Request();
		rq.append('cgi', 'get-tile');
		const xString = x.toString();
		rq.append('x', xString);
		const yString = y.toString();
		rq.append('y', yString);
		const zString = z.toString();
		rq.append('z', zString);
		const colorScaleString = colorScale.toString();
		rq.append('colorscale', colorScaleString);

		/*
		 * Use session token.
		 */
		if (token !== null) {
			rq.append('token', token);
		}

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
	this.fetchTiles = function(token, tileIds, callback) {

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
				self.fetchTile(token, currentTile, internalCallback);
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
	this.updateMap = function(token, xres, yres, xpos, ypos, zoom, mintime, maxtime, useOSMTiles, colorScale, spread, fgColor) {
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

		/*
		 * Use spread.
		 */
		if (spread !== null) {
			const spreadString = spread.toString();
			rq.append('spread', spreadString);
		}

		/*
		 * Use fgColor.
		 */
		if (fgColor !== null) {
			const fgColorString = fgColor.toString();
			rq.append('fgcolor', fgColorString);
		}

		/*
		 * Use session token.
		 */
		if (token !== null) {
			rq.append('token', token);
		}

		const cvs = document.getElementById('map_canvas');
		const idRequest = storage.get(cvs, 'imageRequestId');
		const currentRequestId = idRequest + 1;
		storage.put(cvs, 'imageRequestId', currentRequestId);
		const currentRequestIdString = currentRequestId.toString();
		rq.append('rqid', currentRequestIdString);
		storage.put(cvs, 'osmTiles', []);
		const cgi = globals.cgi;
		const data = rq.getData();

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

				/*
				 * Draw image if it was retrieved.
				 */
				if (img !== null) {
					ctx.drawImage(img, 0, 0);
				}

				/*
				 * Check if we should use OSM tiles.
				 */
				if (useOSMTiles & (colorScale !== '0')) {
					const tileIds = self.calculateTiles(xres, yres, zoom, xpos, ypos, colorScale);
					storage.put(cvs, 'osmTiles', tileIds);

					/*
					 * Internal callback necessary to have
					 * "this" reference.
					 */
					const internalCallback = function() {
						self.updateTiles();
					};

					self.fetchTiles(token, tileIds, internalCallback);
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
		const token = storage.get(cvs, 'token');
		const width = cvs.scrollWidth;
		const height = cvs.scrollHeight;
		const posX = storage.get(cvs, 'posX');
		const posY = storage.get(cvs, 'posY');
		const zoom = storage.get(cvs, 'zoomLevel');
		const timeMin = storage.get(cvs, 'minTime');
		const timeMax = storage.get(cvs, 'maxTime');
		const useOSMTiles = storage.get(cvs, 'useOSMTiles');
		const colorScale = storage.get(cvs, 'colorScale');
		const spread = storage.get(cvs, 'spread');
		const fgColor = storage.get(cvs, 'fgColor');
		ui.updateMap(token, width, height, posX, posY, zoom, timeMin, timeMax, useOSMTiles, colorScale, spread, fgColor);
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
	 * This is called after the user authenticated successfully.
	 */
	this.loginSuccessful = function(token) {
		const cvs = document.getElementById('map_canvas');
		storage.put(cvs, 'token', token);
		ui.hideLogin();
		cvs.style.display = 'block';
		self.refresh();
	};

	/*
	 * This is called when the user clicks on the 'Activities' button.
	 */
	this.showActivities = function() {
		const cvs = document.getElementById('map_canvas');
		const token = storage.get(cvs, 'token');
		const cgi = globals.cgi;
		const request = new Request();
		request.append('cgi', 'get-activities');
		request.append('token', token);
		const data = request.getData();
		const mime = globals.mimeDefault;

		/*
		 * This is called when the server returns activities.
		 */
		const callback = function(content) {
			const response = helper.parseJSON(content);
			const div = document.getElementById('activities');
			ui.displayActivities(div, response);
		};

		ajax.request('POST', cgi, data, mime, callback, false);
	};

	/*
	 * This is called when the user clicks on the 'GeoDB' button.
	 */
	this.showGeoDB = function() {
		const cvs = document.getElementById('map_canvas');
		const token = storage.get(cvs, 'token');
		const cgi = globals.cgi;
		const request = new Request();
		request.append('cgi', 'get-geodb-stats');
		request.append('token', token);
		const data = request.getData();
		const mime = globals.mimeDefault;

		/*
		 * This is called when the server returns activities.
		 */
		const callback = function(content) {
			const response = helper.parseJSON(content);
			const div = document.getElementById('geodb');
			ui.displayGeoDBStats(div, response);
		};

		ajax.request('POST', cgi, data, mime, callback, false);
	};

	/*
	 * This is called when the user clicks on the 'Logout' button.
	 */
	this.logout = function() {
		const cvs = document.getElementById('map_canvas');
		const token = storage.get(cvs, 'token');
		const cgi = globals.cgi;
		const request = new Request();
		request.append('cgi', 'auth-logout');
		request.append('token', token);
		const data = request.getData();
		const mime = globals.mimeDefault;

		/*
		 * This is called when the server confirms the logout.
		 */
		const callback = function(content) {
			const response = helper.parseJSON(content);
			const success = response.Success;

			/*
			 * Check if logout was successful.
			 */
			if (success === true) {
				storage.put(cvs, 'token', null);
				ui.showLogin();
				self.refresh();
			}

		};

		ajax.request('POST', cgi, data, mime, callback, false);
	}

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
		storage.put(cvs, 'spread', '0');
		storage.put(cvs, 'colorScale', '5');
		storage.put(cvs, 'fgColor', null);
		storage.put(cvs, 'minTime', null);
		storage.put(cvs, 'maxTime', null);
		storage.put(cvs, 'useOSMTiles', true);
		storage.put(cvs, 'imageRequestId', 0);
		storage.put(cvs, 'imageResponseId', 0);
		storage.put(cvs, 'imageZoom', 0);
		storage.put(cvs, 'mouseStartX', 0);
		storage.put(cvs, 'mouseStartY', 0);
		storage.put(cvs, 'token', null);
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
		ui.initializeSidebar();
		ui.initializeLogin(self.loginSuccessful);
		helper.blockSite(false);
	};

}

/*
 * The (global) event handlers.
 */
const handler = new Handler();
document.addEventListener('DOMContentLoaded', handler.initialize);

