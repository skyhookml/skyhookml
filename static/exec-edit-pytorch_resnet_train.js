import PytorchTrainGeneric from './exec-edit-pytorch_train-generic.js';
export default PytorchTrainGeneric({
	disabled: ['model', 'dataset'],
	created: function() {
		if(!('Mode' in this.params)) this.$set(this.params, 'Mode', 'yolov3');
		if(!('Width' in this.params)) this.$set(this.params, 'Width', 224);
		if(!('Height' in this.params)) this.$set(this.params, 'Height', 224);
		if(!('NumClasses' in this.params)) this.$set(this.params, 'NumClasses', 0);
		if(!('ValPercent' in this.params)) this.$set(this.params, 'ValPercent', 20);
	},
	basicTemplate: `
<div class="small-container">
	<div class="row mb-2">
		<label class="col-sm-4 col-form-label">Mode</label>
		<div class="col-sm-8">
			<select v-model="params.Mode" class="form-select">
				<option value="resnet18">Resnet18</option>
				<option value="resnet34">Resnet34</option>
				<option value="resnet50">Resnet50</option>
				<option value="resnet101">Resnet101</option>
				<option value="resnet152">Resnet152</option>
			</select>
			<small class="form-text text-muted">
				Select a model architecture. For example, Resnet34 consists of 34 layers, and is suitable for small to medium sized datasets.
			</small>
		</div>
	</div>
	<div class="row mb-2">
		<label class="col-sm-4 col-form-label">Width</label>
		<div class="col-sm-8">
			<input v-model.number="params.Width" type="text" class="form-control">
			<small class="form-text text-muted">
				Resize the image to this width (must be at least 224). Leave as 0 to use the input image without resizing.
			</small>
		</div>
	</div>
	<div class="row mb-2">
		<label class="col-sm-4 col-form-label">Height</label>
		<div class="col-sm-8">
			<input v-model.number="params.Height" type="text" class="form-control">
			<small class="form-text text-muted">
				Resize the image to this height (must be at least 224). Leave as 0 to use the input image without resizing.
			</small>
		</div>
	</div>
	<div class="row mb-2">
		<label class="col-sm-4 col-form-label">Number of Classes</label>
		<div class="col-sm-8">
			<input v-model.number="params.NumClasses" type="text" class="form-control">
			<small class="form-text text-muted">
				The number of image classification categories, or 0 to take it from the label dataset metadata.
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
