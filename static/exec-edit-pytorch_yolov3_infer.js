import utils from './utils.js';

export default {
	data: function() {
		return {
			node: null,
			params: {
				Width: 0,
				Height: 0,
				ConfidenceThreshold: 0,
			},
		};
	},
	created: function() {
		const nodeID = this.$route.params.nodeid;
		utils.request(this, 'GET', '/exec-nodes/'+nodeID, null, (node) => {
			this.node = node;
			try {
				let s = JSON.parse(this.node.Params);
				this.params = s;
			} catch(e) {}
		});
	},
	methods: {
		save: function() {
			utils.request(this, 'POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
				Params: JSON.stringify(this.params),
			}), () => {
				this.$router.push('/ws/'+this.$route.params.ws+'/queries');
			});
		},
	},
	template: `
<div class="small-container m-2">
	<template v-if="node != null">
		<div class="row mb-2">
			<label class="col-sm-2 col-form-label">Width</label>
			<div class="col-sm-10">
				<input v-model.number="params.Width" type="text" class="form-control">
				<small class="form-text text-muted">
					Resize the image to this width. Leave as 0 to use the input image without resizing.
				</small>
			</div>
		</div>
		<div class="row mb-2">
			<label class="col-sm-2 col-form-label">Height</label>
			<div class="col-sm-10">
				<input v-model.number="params.Height" type="text" class="form-control">
				<small class="form-text text-muted">
					Resize the image to this height. Leave as 0 to use the input image without resizing.
				</small>
			</div>
		</div>
		<div class="row mb-2">
			<label class="col-sm-2 col-form-label">Confidence Threshold</label>
			<div class="col-sm-10">
				<input v-model.number="params.ConfidenceThreshold" type="text" class="form-control">
				<small class="form-text text-muted">
					Only output detections with confidence score above this threshold.
				</small>
			</div>
		</div>
		<button v-on:click="save" type="button" class="btn btn-primary">Save</button>
	</template>
</div>
	`,
};
