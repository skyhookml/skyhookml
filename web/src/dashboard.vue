<template>
<div class="flex-container">
	<div class="flex-content-big">
		<h3 class="my-2">Quickstart</h3>
		<div class="card my-2" style="max-width: 800px" role="button">
			<router-link tag="div" :to="'/ws/'+$route.params.ws+'/quickstart/import'">
				<div class="card-body">
					<h5 class="card-title">Import Data</h5>
					<p class="card-text">Import data into SkyhookML, whether it's unlabeled image or video, or datasets with annotations.</p>
				</div>
			</router-link>
		</div>
		<div class="card my-2" style="max-width: 800px" role="button">
			<router-link tag="div" :to="'/ws/'+$route.params.ws+'/annotate-add'">
				<div class="card-body">
					<h5 class="card-title">Annotate</h5>
					<p class="card-text">Label images or videos with object detections, image classes, etc.</p>
				</div>
			</router-link>
		</div>
		<div class="card my-2" style="max-width: 800px" role="button">
			<router-link tag="div" :to="'/ws/'+$route.params.ws+'/quickstart/train'">
				<div class="card-body">
					<h5 class="card-title">Train a Model</h5>
					<p class="card-text">Train a model on labeled datasets.</p>
				</div>
			</router-link>
		</div>
		<div class="card my-2" style="max-width: 800px" role="button">
			<router-link tag="div" :to="'/ws/'+$route.params.ws+'/quickstart/apply'">
				<div class="card-body">
					<h5 class="card-title">Apply a Model</h5>
					<p class="card-text">Apply a pre-trained model or a model that you've trained on new images or videos.</p>
				</div>
			</router-link>
		</div>
	</div>
</div>
</template>

<script>
import utils from './utils.js';

export default {
};
</script>