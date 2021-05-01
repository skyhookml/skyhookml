<template>
<div class="small-container m-2">
	<template v-if="node != null">
		<select-input-size v-model="params.Resize"></select-input-size>
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
</template>

<script>
import utils from '../utils.js';
import SelectInputSize from './select-input-size.vue';

export default {
	components: {
		'select-input-size': SelectInputSize,
	},
	data: function() {
		return {
			params: null,
		};
	},
	props: ['node'],
	created: function() {
		let params = {};
		try {
			let s = JSON.parse(this.node.Params);
			params = s;
		} catch(e) {}
		if(!('Resize' in params)) {
			params.Resize = {
				Mode: 'scale-down',
				MaxDimension: 640,
				Width: 416,
				Height: 416,
				Multiple: 32,
			};
		}
		if(!('ConfidenceThreshold' in params)) params.ConfidenceThreshold = 0.1;
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
