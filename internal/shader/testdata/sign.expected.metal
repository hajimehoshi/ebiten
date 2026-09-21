int F0(bool front_facing, int l0);
int2 F1(bool front_facing, int2 l0);
float F2(bool front_facing, float l0);
float3 F3(bool front_facing, float3 l0);

int F0(bool front_facing, int l0) {
	return sign(l0);
}

int2 F1(bool front_facing, int2 l0) {
	return sign(l0);
}

float F2(bool front_facing, float l0) {
	return sign(l0);
}

float3 F3(bool front_facing, float3 l0) {
	return sign(l0);
}
