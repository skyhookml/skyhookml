<template>
<div class="small-container m-2">
	<template v-if="node != null">
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Length</label>
			<div class="col-sm-10">
				<input v-model="length" type="text" class="form-control">
				<small class="form-text text-muted">
					The number of video frames to extract for each sample.
					To sample images, set length to 1.
					When length is greater than 1, this operation outputs video segments.
				</small>
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Count</label>
			<div class="col-sm-10">
				<input v-model="count" type="text" class="form-control">
				<small class="form-text text-muted">
					The number of samples to extract from the input dataset.
				</small>
			</div>
		</div>
			<div class="form-group row">
				<label class="col-sm-2 col-form-label">Mode</label>
				<div class="col-sm-10">
					<div class="form-check">
						<input class="form-check-input" type="radio" v-model="mode" value="uniform">
						<label class="form-check-label">Uniform: extract samples at a constant framerate.</label>
					</div>
					<div class="form-check">
						<input class="form-check-input" type="radio" v-model="mode" value="random">
						<label class="form-check-label">Random: extract samples randomly.</label>
					</div>
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
			mode: 'uniform',
			length: 1,
			count: 1000,
		};
	},
	props: ['node'],
	created: function() {
		try {
			let s = JSON.parse(this.node.Params);
			this.mode = s.Mode;
			this.length = s.Length;
			this.count = s.Count;
		} catch(e) {}
	},
	methods: {
		save: function() {
			let params = JSON.stringify({
				Mode: this.mode,
				Length: parseInt(this.length),
				Count: parseInt(this.count),
			});
			utils.request(this, 'POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
				Params: params,
			}), () => {
				this.$router.push('/ws/'+this.$route.params.ws+'/pipeline');
			});
		},
	},
};
</script>
