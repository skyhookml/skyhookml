<template>
<div class="small-container">
	<h3>Basics</h3>
	<div class="form-group row">
		<label class="col-sm-4 col-form-label">Learning Rate</label>
		<div class="col-sm-8">
			<input v-model.number="p.LearningRate" type="text" class="form-control" @change="update">
		</div>
	</div>
	<div class="form-group row">
		<label class="col-sm-4 col-form-label">Optimizer</label>
		<div class="col-sm-8">
			<select v-model="p.Optimizer" class="form-select" @change="update">
				<option value="adam">Adam</option>
			</select>
		</div>
	</div>
	<div class="form-group row">
		<label class="col-sm-4 col-form-label">Batch Size</label>
		<div class="col-sm-8">
			<input v-model.number="p.BatchSize" type="text" class="form-control" @change="update">
		</div>
	</div>
	<div class="form-group row">
		<label class="col-sm-4 col-form-label">Auto Batch Size</label>
		<div class="col-sm-8">
			<div class="form-check">
				<input class="form-check-input" type="checkbox" v-model="p.AutoBatchSize" @change="update">
				<label class="form-check-label">
					Automatically reduce the batch size if we run out of GPU memory.
				</label>
			</div>
		</div>
	</div>

	<h3>Stop Condition</h3>
	<div class="form-group row">
		<label class="col-sm-4 col-form-label">Max Epochs</label>
		<div class="col-sm-8">
			<input v-model.number="p.StopCondition.MaxEpochs" type="text" class="form-control" @change="update">
			<small class="form-text text-muted">
				Stop training after this many epochs. Leave 0 to disable this stop condition.
			</small>
		</div>
	</div>
	<div class="form-group row">
		<label class="col-sm-4 col-form-label">Epochs Without Improvement</label>
		<div class="col-sm-8">
			<input v-model.number="p.StopCondition.ScoreMaxEpochs" type="text" class="form-control" @change="update">
			<small class="form-text text-muted">
				Stop training if this many epochs have elapsed without non-negligible improvement in the score. Leave 0 to disable this stop condition.
			</small>
		</div>
	</div>
	<div class="form-group row">
		<label class="col-sm-4 col-form-label">Improvement Threshold</label>
		<div class="col-sm-8">
			<input v-model.number="p.StopCondition.ScoreEpsilon" type="text" class="form-control" @change="update">
			<small class="form-text text-muted">
				Increases in the score less than this threshold are considered negligible. 0 implies that any increase will reset the timer for Epochs Without Improvement.
			</small>
		</div>
	</div>

	<h3>Model Saver</h3>
	<div class="form-group row">
		<label class="col-sm-4 col-form-label">Saver Mode</label>
		<div class="col-sm-8">
			<select v-model="p.ModelSaver.Mode" class="form-select" @change="update">
				<option value="latest">Save the latest model</option>
				<option value="best">Save the model with best validation score</option>
			</select>
		</div>
	</div>

	<h3>Rate Decay</h3>
	<div class="form-group row">
		<label class="col-sm-4 col-form-label">Rate Decay Mode</label>
		<div class="col-sm-8">
			<select v-model="p.RateDecay.Op" class="form-select" @change="update">
				<option value="">None (constant learning rate)</option>
				<option value="step">Step (reduce rate by a factor every few epochs)</option>
				<option value="plateau">Plateau (reduce rate if score isn't improving)</option>
			</select>
		</div>
	</div>
	<template v-if="p.RateDecay.Op == 'step'">
		<div class="form-group row">
			<label class="col-sm-4 col-form-label">Step Size</label>
			<div class="col-sm-8">
				<input v-model.number="p.RateDecay.StepSize" type="text" class="form-control" @change="update">
				<small class="form-text text-muted">
					Reduce learning rate each time this many epochs have elapsed.
				</small>
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-4 col-form-label">Gamma</label>
			<div class="col-sm-8">
				<input v-model.number="p.RateDecay.StepGamma" type="text" class="form-control" @change="update">
				<small class="form-text text-muted">
					To reduce learning rate, multiply it by this factor.
				</small>
			</div>
		</div>
	</template>
	<template v-if="p.RateDecay.Op == 'plateau'">
		<div class="form-group row">
			<label class="col-sm-4 col-form-label">Gamma</label>
			<div class="col-sm-8">
				<input v-model.number="p.RateDecay.PlateauFactor" type="text" class="form-control" @change="update">
				<small class="form-text text-muted">
					Multiply the learning rate by this factor when a plateau is detected.
				</small>
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-4 col-form-label">Patience</label>
			<div class="col-sm-8">
				<input v-model.number="p.RateDecay.PlateauPatience" type="text" class="form-control" @change="update">
				<small class="form-text text-muted">
					Wait for no improvement for this many epochs before reducing the learning rate.
				</small>
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-4 col-form-label">Plateau Epsilon</label>
			<div class="col-sm-8">
				<input v-model.number="p.RateDecay.PlateauThreshold" type="text" class="form-control" @change="update">
				<small class="form-text text-muted">
					Score improvements less than this threshold are still considered a plateau.
				</small>
			</div>
		</div>
		<div class="form-group row">
			<label class="col-sm-4 col-form-label">Minimum Learning Rate</label>
			<div class="col-sm-8">
				<input v-model.number="p.RateDecay.PlateauMin" type="text" class="form-control" @change="update">
				<small class="form-text text-muted">
					Don't reduce the learning rate below this value.
				</small>
			</div>
		</div>
	</template>
</div>
</template>

<script>
import utils from '../utils.js';

export default {
	data: function() {
		return {
			p: {},
		};
	},
	props: ['node', 'value'],
	created: function() {
		let p = {};
		try {
			let s = JSON.parse(this.value);
			p = s;
		} catch(e) {}
		if(!p.LearningRate) p.LearningRate = 0.001;
		if(!p.Optimizer) p.Optimizer = 'adam';
		if(!p.BatchSize) p.BatchSize = 1;
		if(!p.AutoBatchSize) p.AutoBatchSize = true;
		if(!p.StopCondition) p.StopCondition = {};
		if(!p.StopCondition.MaxEpochs) p.StopCondition.MaxEpochs = 0;
		if(!p.StopCondition.ScoreEpsilon) p.StopCondition.ScoreEpsilon = 0;
		if(!p.StopCondition.ScoreMaxEpochs) p.StopCondition.ScoreMaxEpochs = 25;
		if(!p.ModelSaver) p.ModelSaver = {};
		if(!p.ModelSaver.Mode) p.ModelSaver.Mode = 'best';
		if(!p.RateDecay) p.RateDecay = {};
		if(!p.RateDecay.Op) p.RateDecay.Op = '';
		if(!p.RateDecay.StepSize) p.RateDecay.StepSize = 1;
		if(!p.RateDecay.StepGamma) p.RateDecay.StepGamma = 0.1;
		if(!p.RateDecay.PlateauFactor) p.RateDecay.PlateauFactor = 0.1;
		if(!p.RateDecay.PlateauPatience) p.RateDecay.PlateauPatience = 10;
		if(!p.RateDecay.PlateauThreshold) p.RateDecay.PlateauThreshold = 0;
		if(!p.RateDecay.PlateauMin) p.RateDecay.PlateauMin = 0.0001;
		this.p = p;
		this.update();
	},
	methods: {
		update: function() {
			this.$emit('input', JSON.stringify(this.p));
		},
	},
};
</script>