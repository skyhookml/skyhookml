<template>
<div class="small-container m-2">
	<template v-if="node != null">
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">URL</label>
			<div class="col-sm-10">
				<input v-model="params.Source.URL" type="text" class="form-control">
				<small class="form-text text-muted">The URL source for Web-Mercator images, with placeholders for the zoom and position. For example, https://example.com/[ZOOM]/[X]/[Y]?format=jpeg.</small>
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Zoom</label>
			<div class="col-sm-10">
				<input v-model.number="params.Source.Zoom" type="text" class="form-control">
				<small class="form-text text-muted">Desired zoom level. For example, at zoom 18, resolution is roughly 60 cm/pixel.</small>
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Capture Mode</label>
			<div class="col-sm-10">
				<div class="form-check">
					<input class="form-check-input" type="radio" v-model="params.CaptureMode" value="dense">
					<label class="form-check-label">Dense: Capture all Web-Mercator tiles in a given geographical bounding box (rectangle).</label>
				</div>
				<div class="form-check">
					<input class="form-check-input" type="radio" v-model="params.CaptureMode" value="geojson">
					<label class="form-check-label">GeoJSON: Capture images corresponding to objects in a GeoJSON dataset provided as input.</label>
				</div>
				<small class="form-text text-muted">The capture mode specifies how the system should decide which images to extract.</small>
			</div>
		</div>
		<div class="form-group row" v-if="params.CaptureMode == 'geojson'">
			<label class="col-sm-2 col-form-label">Object Mode</label>
			<div class="col-sm-10">
				<div class="form-check">
					<input class="form-check-input" type="radio" v-model="params.ObjectMode" value="centered-all">
					<label class="form-check-label">Centered (all): Create an image centered around each GeoJSON object.</label>
				</div>
				<div class="form-check">
					<input class="form-check-input" type="radio" v-model="params.ObjectMode" value="centered-disjoint">
					<label class="form-check-label">Centered (disjoint): Like Centered(all), but if two images overlap, then only one is used.</label>
				</div>
				<div class="form-check">
					<input class="form-check-input" type="radio" v-model="params.ObjectMode" value="tiles">
					<label class="form-check-label">Tiles: Capture all Web-Mercator tiles that intersect one or more GeoJSON objects.</label>
				</div>
				<small class="form-text text-muted">The object mode specifies how the system should use GeoJSON objects to create images.</small>
			</div>
		</div>
		<div class="form-group row" v-if="params.CaptureMode == 'geojson' && params.ObjectMode == 'tiles'">
			<label class="col-sm-2 col-form-label">Buffer</label>
			<div class="col-sm-10">
				<input v-model.number="params.Buffer" type="text" class="form-control">
				<small class="form-text text-muted">A padding around GeoJSON objects that should be covered by the selected GeoJSON tiles.</small>
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-4 col-form-label">Materialize</label>
			<div class="col-sm-8">
				<div class="form-check">
					<input class="form-check-input" type="checkbox" v-model="params.Materialize">
					<label class="form-check-label">
						Materialize the images in the output dataset (fetch the images immediately).
					</label>
				</div>
				<small class="form-text text-muted">If unchecked, the images will be loaded lazily upon access.</small>
			</div>
		</div>

		<!-- Currently, ImageDims is ignored unless capturing images centered around GeoJSON objects. -->
		<template v-if="params.CaptureMode == 'geojson' && (params.ObjectMode == 'centered-all' || params.ObjectMode == 'centered-disjoint')">
			<h3>Image Dimensions</h3>
			<div class="form-group row">
				<label class="col-sm-2 col-form-label">Image Width</label>
				<div class="col-sm-10">
					<input v-model.number="params.ImageDims[0]" type="text" class="form-control">
					<small class="form-text text-muted">
						Width of output images. If 0, the width is based on the size of the corresponding GeoJSON object.
					</small>
				</div>
			</div>
			<div class="form-group row">
				<label class="col-sm-2 col-form-label">Image Height</label>
				<div class="col-sm-10">
					<input v-model.number="params.ImageDims[1]" type="text" class="form-control">
					<small class="form-text text-muted">
						Height of output images. If 0, the width is based on the size of the corresponding GeoJSON object.
					</small>
				</div>
			</div>
		</template>

		<template v-if="params.CaptureMode == 'dense'">
			<h3>Bounding Box</h3>
			<div class="form-group row">
				<label class="col-sm-2 col-form-label">Start Longitude</label>
				<div class="col-sm-10">
					<input v-model.number="params.Bbox[0]" type="text" class="form-control">
				</div>
			</div>
			<div class="form-group row">
				<label class="col-sm-2 col-form-label">Start Latitude</label>
				<div class="col-sm-10">
					<input v-model.number="params.Bbox[1]" type="text" class="form-control">
				</div>
			</div>
			<div class="form-group row">
				<label class="col-sm-2 col-form-label">End Longitude</label>
				<div class="col-sm-10">
					<input v-model.number="params.Bbox[2]" type="text" class="form-control">
				</div>
			</div>
			<div class="form-group row">
				<label class="col-sm-2 col-form-label">End Latitude</label>
				<div class="col-sm-10">
					<input v-model.number="params.Bbox[3]" type="text" class="form-control">
				</div>
			</div>
		</template>

		<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
	</template>
</div>
</template>

<script>
import utils from '../utils.js';

export default {
	data: function() {
		return {
			params: null,
		};
	},
	props: ['node'],
	created: function() {
		let params = {};
		try {
			params = JSON.parse(this.node.Params);
		} catch(e) {}
		if(!('Source' in params)) params.Source = {};
		if(!('URL' in params.Source)) params.Source.URL = '';
		if(!('Zoom' in params.Source)) params.Source.Zoom = 17;
		if(!('CaptureMode' in params)) params.CaptureMode = 'dense';
		if(!('ObjectMode' in params)) params.ObjectMode = 'centered-all';
		if(!('Buffer' in params)) params.Buffer = 128;
		if(!('Materialize' in params)) params.Materialize = false;
		if(!('ImageDims' in params)) params.ImageDims = [256, 256];
		if(!('Bbox' in params)) params.Bbox = [0, 0, 0, 0];
		this.params = params;
	},
	methods: {
		save: function() {
			utils.request(this, 'POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
				Params: JSON.stringify(this.params),
			}), () => {
				this.$router.push('/ws/'+this.$route.params.ws+'/pipeline');
			});
		},
	},
};
</script>
