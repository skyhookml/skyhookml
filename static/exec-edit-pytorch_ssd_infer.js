import utils from './utils.js';

export default {
	data: function() {
		return {
			params: null,
		};
	},
	props: ['node'],
	created: function() {
		let params;
		try {
			let s = JSON.parse(this.node.Params);
			params = s;
		} catch(e) {}
		if(!('ConfidenceThreshold' in params)) params.ConfidenceThreshold = 0.1;
		this.params = params;
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
