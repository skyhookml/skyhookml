import PytorchTrainGeneric from './exec-edit-pytorch_train-generic.js';
export default PytorchTrainGeneric({
	disabled: ['model', 'dataset'],
	created: function() {
		if(!('Mode' in this.params)) this.$set(this.params, 'Mode', 'yolov3');
		if(!('Width' in this.params)) this.$set(this.params, 'Width', 0);
		if(!('Height' in this.params)) this.$set(this.params, 'Height', 0);
		if(!('ValPercent' in this.params)) this.$set(this.params, 'ValPercent', 20);
	},
	basicTemplate: `
<div class="small-container">
	<div class="row mb-2">
		<label class="col-sm-4 col-form-label">Mode</label>
		<div class="col-sm-8">
			<select v-model="params.Mode" class="form-select">
				<option value="yolov3">YOLOv3</option>
				<option value="yolov3-spp">YOLOv3-SPP</option>
				<option value="yolov3-tiny">YOLOv3-Tiny</option>
			</select>
			<small class="form-text text-muted">
				YOLOv3 and YOLOv3-SPP are large models providing high accuracy (YOLOv3-SPP may provide slightly higher accuarcy).
				YOLOv3-Tiny is a small model that is fast but provides lower accuracy.
			</small>
		</div>
	</div>
	<div class="row mb-2">
		<label class="col-sm-4 col-form-label">Width</label>
		<div class="col-sm-8">
			<input v-model.number="params.Width" type="text" class="form-control">
			<small class="form-text text-muted">
				Resize the image to this width (must be a multiple of 32). Leave as 0 to use the input image without resizing.
			</small>
		</div>
	</div>
	<div class="row mb-2">
		<label class="col-sm-4 col-form-label">Height</label>
		<div class="col-sm-8">
			<input v-model.number="params.Height" type="text" class="form-control">
			<small class="form-text text-muted">
				Resize the image to this height (must be a multiple of 32). Leave as 0 to use the input image without resizing.
			</small>
		</div>
	</div>
	<div class="row mb-2">
		<label class="col-sm-4 col-form-label">Validation Percentage</label>
		<div class="col-sm-8">
			<input v-model.number="params.ValPercent" type="text" class="form-control">
			<small class="form-text text-muted">
				Use this percentage of the input data for validation. The rest will be used for training.
			</small>
		</div>
	</div>
</div>
	`,
});
