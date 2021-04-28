<template>
<div class="small-container m-2">
	<template v-if="node != null">
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">X Offset</label>
			<div class="col-sm-10">
				<input v-model.number="start[0]" type="text" class="form-control">
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Y Offset</label>
			<div class="col-sm-10">
				<input v-model.number="start[1]" type="text" class="form-control">
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Crop Width</label>
			<div class="col-sm-10">
				<input v-model.number="cropDims[0]" type="text" class="form-control">
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Crop Height</label>
			<div class="col-sm-10">
				<input v-model.number="cropDims[1]" type="text" class="form-control">
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Resize Width</label>
			<div class="col-sm-10">
				<input v-model.number="resizeDims[0]" type="text" class="form-control">
				<small class="form-text text-muted">
					Resize to this width after cropping. Leave 0 to disable.
				</small>
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-2 col-form-label">Resize Height</label>
			<div class="col-sm-10">
				<input v-model.number="resizeDims[1]" type="text" class="form-control">
				<small class="form-text text-muted">
					Resize to this width after cropping. Leave 0 to disable.
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
			start: [0, 0],
			cropDims: [0, 0],
			resizeDims: [0, 0],
		};
	},
	props: ['node'],
	created: function() {
		try {
			let s = JSON.parse(this.node.Params);
			this.start = s.Start;
			this.cropDims = s.CropDims;
			this.resizeDims = s.ResizeDims;
		} catch(e) {}
	},
	methods: {
		save: function() {
			let params = JSON.stringify({
				Start: this.start,
				CropDims: this.cropDims,
				ResizeDims: this.resizeDims,
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