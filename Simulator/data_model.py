import random
import numpy as np
import pandas as pd


class BoilerDataSet(object):
    """
    first run data_preparation.py to generate data.csv
    prepare boiler training and validation dataset
    simple version(small action dimension)

    """
    # @todo 还未归一化
    
    def __init__(self, num_steps, val_ratio=0.1):
        self.num_steps = num_steps  # time steps
        self.val_ratio = val_ratio  # train test ratio
        
        # Read csv file
        self.raw_data = pd.read_csv("./data/sim_train.csv", index_col='时间戳')

        # sort csv file
        cols = self.raw_data.columns.tolist()
        # print("origin len: {0}".format(len(cols)))
        cols = (cols[51:52] + cols[53:59] + cols [60:61] + cols[62:63] + cols[150:152]   # external input 
            + cols[0:50] + cols[52:53] + cols[122:139]  # Coal Pulverizing state
            + cols[50:51] + cols[59:60] + cols[61:62] + cols[63:101] + cols[112:114] + cols[118:122] + cols[139:145] + cols[146:149] + cols[152:158]    # Burning state
            + cols[101:112] + cols[114:118] + cols[145:146] + cols[149:150] # Steam Circulation state
            + cols[158:173] + cols[196:202] # Coal Pulverizing action
            + cols[173:192]                 # Burning action
            + cols[192:196])                # Steam Circulation action
        # print("ordered len: {0}".format(len(cols)))
        self.raw_data = self.raw_data[cols]

        # divide train set and valid set
        self.train_X, self.train_y, self.valid_X, self.valid_y = self.prepare_data(self.raw_data)

    def prepare_data(self, data):
        # split into groups of num_steps

        # Fetch input data with num_steps history
        X = np.array([data.iloc[i: i + self.num_steps].values
                    for i in range(len(data) - self.num_steps)])

        # Fetch output data, predict num_steps value
        # state value only, 0-157 columns are state values
        y = np.array([data.iloc[i + self.num_steps, 0:158].values
                    for i in range(len(data) - self.num_steps)])

        train_size = int(len(X) * (1.0 - self.val_ratio))
        train_X, valid_X = X[:train_size], X[train_size:]
        train_y, valid_y = y[:train_size], y[train_size:]
        return train_X, train_y, valid_X, valid_y

    def generate_one_epoch(self, data_X, data_y, batch_size):
        num_batches = int(len(data_X)) // batch_size
        # if batch_size * num_batches < len(self.train_X):
        #     num_batches += 1

        batch_indices = list(range(num_batches))
        random.shuffle(batch_indices)
        for j in batch_indices:
            batch_X = data_X[j * batch_size: (j + 1) * batch_size]
            batch_y = data_y[j * batch_size: (j + 1) * batch_size]
            yield batch_X, batch_y