<template>
<div class="small-container m-2">
	<template v-if="node != null">
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Denoise Size</label>
			<div class="col-sm-10">
				<input v-model.number="params.DenoiseSize" type="text" class="form-control">
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Grow Size</label>
			<div class="col-sm-10">
				<input v-model.number="params.GrowSize" type="text" class="form-control">
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Simplify Threshold</label>
			<div class="col-sm-10">
				<input v-model.number="params.SimplifyThreshold" type="text" class="form-control">
			</div>
		</div>
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
		if(!('DenoiseSize' in params)) params.DenoiseSize = 5;
		if(!('GrowSize' in params)) params.GrowSize = 5;
		if(!('SimplifyThreshold' in params)) params.SimplifyThreshold = 0.01;
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
