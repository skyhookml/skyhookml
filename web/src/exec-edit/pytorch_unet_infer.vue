<template>
<div class="small-container m-2">
	<template v-if="node != null">
		<div class="row mb-2">
			<label class="col-sm-2 col-form-label">Width</label>
			<div class="col-sm-10">
				<input v-model.number="params.Width" type="text" class="form-control">
				<small class="form-text text-muted">
					Resize the image to this width (must be a multiple of 32). Leave as 0 to use the input image without resizing.
				</small>
			</div>
		</div>
		<div class="row mb-2">
			<label class="col-sm-2 col-form-label">Height</label>
			<div class="col-sm-10">
				<input v-model.number="params.Height" type="text" class="form-control">
				<small class="form-text text-muted">
					Resize the image to this height (must be a multiple of 32). Leave as 0 to use the input image without resizing.
				</small>
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
			let s = JSON.parse(this.node.Params);
			params = s;
		} catch(e) {}
		if(!('Width' in params)) params.Width = 0;
		if(!('Height' in params)) params.Height = 0;
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