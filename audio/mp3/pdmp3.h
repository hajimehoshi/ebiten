#ifndef PDMP3_H
#define PDMP3_H

#include <stddef.h>
#include <stdint.h>

//#define DEBUG
#define IMDCT_TABLES
#define IMDCT_NTABLES
#define POW34_ITERATE
#define POW34_TABLE
#define OUTPUT_RAW

#define OK    0
#define ERROR -1
#define TRUE       1
#define FALSE      0

/* Types used in the frame header */
typedef enum { /* Layer number */
  mpeg1_layer_reserved = 0,
  mpeg1_layer_3        = 1,
  mpeg1_layer_2        = 2,
  mpeg1_layer_1        = 3
}
t_mpeg1_layer;
typedef enum { /* Modes */
  mpeg1_mode_stereo = 0,
  mpeg1_mode_joint_stereo,
  mpeg1_mode_dual_channel,
  mpeg1_mode_single_channel
}
t_mpeg1_mode;

typedef struct { /* MPEG1 Layer 1-3 frame header */
  unsigned id;                 /* 1 bit */
  t_mpeg1_layer layer;         /* 2 bits */
  unsigned protection_bit;     /* 1 bit */
  unsigned bitrate_index;      /* 4 bits */
  unsigned sampling_frequency; /* 2 bits */
  unsigned padding_bit;        /* 1 bit */
  unsigned private_bit;        /* 1 bit */
  t_mpeg1_mode mode;           /* 2 bits */
  unsigned mode_extension;     /* 2 bits */
  unsigned copyright;          /* 1 bit */
  unsigned original_or_copy;   /* 1 bit */
  unsigned emphasis;           /* 2 bits */
}
t_mpeg1_header;
typedef struct {  /* MPEG1 Layer 3 Side Information : [2][2] means [gr][ch] */
  unsigned main_data_begin;         /* 9 bits */
  unsigned private_bits;            /* 3 bits in mono,5 in stereo */
  unsigned scfsi[2][4];             /* 1 bit */
  unsigned part2_3_length[2][2];    /* 12 bits */
  unsigned big_values[2][2];        /* 9 bits */
  unsigned global_gain[2][2];       /* 8 bits */
  unsigned scalefac_compress[2][2]; /* 4 bits */
  unsigned win_switch_flag[2][2];   /* 1 bit */
  /* if(win_switch_flag[][]) */ //use a union dammit
  unsigned block_type[2][2];        /* 2 bits */
  unsigned mixed_block_flag[2][2];  /* 1 bit */
  unsigned table_select[2][2][3];   /* 5 bits */
  unsigned subblock_gain[2][2][3];  /* 3 bits */
  /* else */
  /* table_select[][][] */
  unsigned region0_count[2][2];     /* 4 bits */
  unsigned region1_count[2][2];     /* 3 bits */
  /* end */
  unsigned preflag[2][2];           /* 1 bit */
  unsigned scalefac_scale[2][2];    /* 1 bit */
  unsigned count1table_select[2][2];/* 1 bit */
  unsigned count1[2][2];            /* Not in file,calc. by huff.dec.! */
}
t_mpeg1_side_info;
typedef struct { /* MPEG1 Layer 3 Main Data */
  unsigned  scalefac_l[2][2][21];    /* 0-4 bits */
  unsigned  scalefac_s[2][2][12][3]; /* 0-4 bits */
  float is[2][2][576];               /* Huffman coded freq. lines */
}
t_mpeg1_main_data;
typedef struct { /* Scale factor band indices,for long and short windows */
  unsigned l[23];
  unsigned s[14];
}
t_sf_band_indices;

unsigned Get_Byte(void);
unsigned Get_Bytes(unsigned num, unsigned* data_vec);
unsigned Get_Filepos(void);
int Decode_L3(void);
int Read_Frame(void);
size_t writeToWriter(void* data, int size);

unsigned Get_Main_Pos(void);
int Set_Main_Pos(unsigned bit_pos);

int Read_Main_L3(void);
int Read_Audio_L3(void);
static int Read_Header(void);
void Read_Huffman(unsigned part_2_start,unsigned gr,unsigned ch);

void L3_Requantize(unsigned gr,unsigned ch);
void L3_Reorder(unsigned gr,unsigned ch);
void L3_Stereo(unsigned gr);
void L3_Antialias(unsigned gr,unsigned ch);
void L3_Hybrid_Synthesis(unsigned gr,unsigned ch);
void L3_Frequency_Inversion(unsigned gr,unsigned ch);
void L3_Subband_Synthesis(unsigned gr,unsigned ch, unsigned* outdata);

void Requantize_Process_Long(unsigned gr,unsigned ch,unsigned is_pos,unsigned sfb);

int Read_CRC(void);

int Huffman_Decode(unsigned table_num, int32_t* x, int32_t*y, int32_t* v, int32_t* w);

#endif
