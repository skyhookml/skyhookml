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
		if(!('URL' in params)) params.URL = '';
		if(!('Zoom' in params)) params.Zoom = 17;
		if(!('Buffer' in params)) params.Buffer = 128;
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
	template: `
<div class="small-container m-2">
	<template v-if="node != null">
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">URL</label>
			<div class="col-sm-10">
				<input v-model="params.URL" type="text" class="form-control">
				<small class="form-text text-muted">The URL source for Web-Mercator images, with placeholders for the zoom and position. For example, https://example.com/[ZOOM]/[X]/[Y]?format=jpeg.</small>
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Zoom</label>
			<div class="col-sm-10">
				<input v-model.number="params.Zoom" type="text" class="form-control">
				<small class="form-text text-muted">Desired zoom level. For example, at zoom 18, resolution is roughly 60 cm/pixel.</small>
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Buffer</label>
			<div class="col-sm-10">
				<input v-model.number="params.Buffer" type="text" class="form-control">
				<small class="form-text text-muted">The buffer (in pixels) added to the region of interest (RoI).  </small>
			</div>
		</div>
		<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
	</template>
</div>
	`,
};
