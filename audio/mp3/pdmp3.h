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

#endif
